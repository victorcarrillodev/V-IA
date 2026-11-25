// Package review ejecuta las comprobaciones del proyecto y emite un recibo ligado al cambio exacto
// (dos árboles git congelados). La revisión independiente es por lente (4R), con veredicto que
// calcula el código —no el revisor—, una sola corrección acotada y escalado a decisión humana.
package review

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"
)

type Check struct {
	Cmd    string `json:"cmd"`
	Exit   int    `json:"exit"`
	Output string `json:"output_tail"`
}

// Estados del recibo.
const (
	StateFailed             = "failed"              // alguna comprobación falló
	StateStale              = "stale"               // el árbol cambió mientras corrían las comprobaciones
	StateReady              = "ready"               // pasó y no exige revisión
	StateChecking           = "checking"            // pasó y faltan revisiones de lente
	StateCorrectionRequired = "correction_required" // un revisor encontró un bloqueante causado por el cambio
	StateApproved           = "approved"            // todas las lentes aprobaron
	StateEscalated          = "escalated"           // la única corrección no bastó: decide una persona
)

// Estados públicos (vocabulario del orquestador).
const (
	PublicWorking       = "working"
	PublicChecking      = "checking"
	PublicReady         = "ready"
	PublicNeedsDecision = "needs_decision"
)

// Finding es un hallazgo de un revisor. Solo bloquean BLOCKER/CRITICAL causados por el cambio.
type Finding struct {
	Location          string   `json:"location"`
	Severity          string   `json:"severity"` // BLOCKER | CRITICAL | WARNING | SUGGESTION
	Claim             string   `json:"claim"`
	EvidenceClass     string   `json:"evidence_class,omitempty"`
	CausalDisposition string   `json:"causal_disposition,omitempty"` // introduced | behavior-activated | worsened | pre-existing | base-only | unknown
	ProofRefs         []string `json:"proof_refs,omitempty"`
}

// Inspection declara qué miró el revisor: debe cubrir todo el manifiesto de rutas.
type Inspection struct {
	Status string   `json:"status"` // complete
	Paths  []string `json:"paths"`
}

// ReviewInput es lo que entrega un revisor de una lente (JSON estricto).
type ReviewInput struct {
	CandidateID string     `json:"candidate_id"`
	Lens        string     `json:"lens"`
	Reviewer    string     `json:"reviewer"`
	Inspection  Inspection `json:"inspection"`
	Findings    []Finding  `json:"findings"`
	Evidence    []string   `json:"evidence"`
}

// LensReview es una revisión admitida; el veredicto lo calcula V-IA.
type LensReview struct {
	Lens      string    `json:"lens"`
	Reviewer  string    `json:"reviewer"`
	Inspected int       `json:"inspected_paths"`
	Findings  []Finding `json:"findings"`
	Evidence  []string  `json:"evidence"`
	Verdict   string    `json:"verdict"` // approve | changes
	At        string    `json:"at"`
}

type Receipt struct {
	CandidateID   string     `json:"candidate_id"`
	Base          string     `json:"base,omitempty"` // referencia pedida con --base; "" = HEAD
	BaseTree      string     `json:"base_tree,omitempty"`
	CandidateTree string     `json:"candidate_tree,omitempty"`
	Paths         []string   `json:"paths,omitempty"`
	Created       string     `json:"created"`
	Result        string     `json:"result"` // pass | fail | stale
	Checks        []Check    `json:"checks"`
	Risk          Assessment `json:"risk"`
	// Congelado al emitir el recibo: ni el agente ni un revisor lo reescriben después.
	SelectedLenses []string     `json:"selected_lenses,omitempty"`
	LensReason     string       `json:"lens_selection_reason,omitempty"`
	Reviews        []LensReview `json:"reviews,omitempty"`
	Attempt        int          `json:"attempt"` // 1 = revisión inicial; 2 = tras la única corrección
	Refutation     *Refutation  `json:"refutation,omitempty"`
	// BlockedTrees son los árboles que un revisor bloqueó en esta revisión.
	BlockedTrees []string `json:"blocked_trees,omitempty"`
	// Request es la petición literal: el revisor contrasta contra ella.
	Request string `json:"request,omitempty"`
	State   string `json:"state"`
}

// RunOptions son las elecciones del agente que se congelan en el recibo.
type RunOptions struct {
	Base       string
	Lenses     []string
	LensReason string
	EscItem    int
	EscReason  string
	Request    string
	Timeout    time.Duration
}

func receiptPath(root string) string { return filepath.Join(root, ".v-ia", "receipt.json") }

// DetectCommands decide qué comprobar: .v-ia/checks.txt manda; si no, se infiere.
func DetectCommands(root string) []string {
	if f, err := os.Open(filepath.Join(root, ".v-ia", "checks.txt")); err == nil {
		defer f.Close()
		var out []string
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			if l := strings.TrimSpace(sc.Text()); l != "" && !strings.HasPrefix(l, "#") {
				out = append(out, l)
			}
		}
		return out
	}
	var out []string
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
		out = append(out, "go build ./...", "go vet ./...", "go test ./...")
	}
	if b, err := os.ReadFile(filepath.Join(root, "package.json")); err == nil && strings.Contains(string(b), `"test"`) {
		out = append(out, "npm test --silent")
	}
	return out
}

func tail(s string, n int) string {
	if len(s) > n {
		return "…" + s[len(s)-n:]
	}
	return s
}

// Run ejecuta las comprobaciones y guarda el recibo. Sin comprobaciones => error: un recibo vacío
// no demuestra nada. El candidato se congela antes y se vuelve a medir después: si cambió, el
// recibo es "stale" y no vale.
func Run(root string, cmds []string, o RunOptions) (Receipt, error) {
	if len(cmds) == 0 {
		return Receipt{}, errors.New("no hay comprobaciones: añade comandos en .v-ia/checks.txt (uno por línea)")
	}
	if o.Timeout == 0 {
		o.Timeout = 10 * time.Minute
	}
	prev, prevErr := load(root)
	if prevErr == nil && prev.State == StateEscalated {
		return Receipt{}, errors.New("la revisión anterior quedó ESCALADA (la única corrección no bastó): decide cómo seguir y ejecuta 'v-ia review reset'")
	}
	if len(o.Request) > maxRequest {
		return Receipt{}, fmt.Errorf("la petición supera %d bytes: resúmela en la ficha y cita lo esencial", maxRequest)
	}
	// dentro de una misma revisión (correcciones incluidas) la petición queda congelada
	if prevErr == nil && prev.State != StateReady && prev.State != StateApproved && prev.Request != "" {
		if o.Request != "" && o.Request != prev.Request {
			return Receipt{}, errors.New("la petición está congelada en esta revisión: para cambiarla ejecuta 'v-ia review reset'")
		}
		o.Request = prev.Request
	}
	cand, cerr := TakeCandidate(root, o.Base)
	r := Receipt{Result: "pass", Attempt: 1, Base: o.Base, Request: o.Request}
	if prevErr == nil {
		switch {
		case prev.State == StateCorrectionRequired:
			r.Attempt = prev.Attempt + 1 // esta es la corrección acotada
		case prev.State != StateReady && prev.State != StateApproved:
			r.Attempt = prev.Attempt // una corrección que falla no devuelve el presupuesto
		}
	}
	if prevErr == nil && prev.State != StateReady && prev.State != StateApproved {
		blockedTrees := prev.BlockedTrees
		if prev.State == StateCorrectionRequired {
			blockedTrees = append(blockedTrees, prev.CandidateTree)
		}
		if cerr == nil && slices.Contains(blockedTrees, cand.CandidateTree) {
			return Receipt{}, errors.New("no hay corrección: ese código ya fue bloqueado en esta revisión")
		}
		r.BlockedTrees = blockedTrees
	}
	for _, c := range cmds {
		ctx, cancel := context.WithTimeout(context.Background(), o.Timeout)
		cmd := shellCommand(ctx, c)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		cancel()
		ck := Check{Cmd: c, Output: tail(string(out), 1500)}
		if err != nil {
			ck.Exit = 1
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				ck.Exit = ee.ExitCode()
			}
			r.Result = "fail"
		}
		r.Checks = append(r.Checks, ck)
	}
	if cerr == nil {
		after, err := TakeCandidate(root, o.Base)
		if err != nil || after.ID != cand.ID {
			r.Result = "stale"
		}
		r.CandidateID, r.BaseTree, r.CandidateTree, r.Paths = cand.ID, cand.BaseTree, cand.CandidateTree, cand.Paths
		r.Risk = AssessCandidate(root, cand)
	} else {
		id, err := currentID(root, o.Base)
		if err != nil {
			return r, err
		}
		r.CandidateID = id
		r.Risk = unknown("no se pudo congelar el candidato: " + cerr.Error())
	}
	if o.EscItem != 0 || o.EscReason != "" {
		if err := r.Risk.Escalate(o.EscItem, o.EscReason); err != nil {
			return r, err
		}
	}
	lenses, err := SelectLenses(r.Risk.Tier, o.Lenses, o.LensReason)
	if err != nil {
		return r, err
	}
	r.SelectedLenses, r.LensReason = lenses, o.LensReason
	r.Created = time.Now().UTC().Format(time.RFC3339)
	r.State = initialState(r)
	return r, save(root, r)
}

func initialState(r Receipt) string {
	switch {
	case r.Result == "fail":
		return StateFailed
	case r.Result == "stale":
		return StateStale
	case r.Risk.ReviewDue && len(r.SelectedLenses) > 0:
		return StateChecking
	}
	return StateReady
}

func save(root string, r Receipt) error {
	b, _ := json.MarshalIndent(r, "", "  ")
	if err := os.MkdirAll(filepath.Dir(receiptPath(root)), 0o755); err != nil {
		return err
	}
	tmp := receiptPath(root) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, receiptPath(root)) // atómico: nadie lee un recibo a medias
}

func load(root string) (Receipt, error) {
	var r Receipt
	raw, err := os.ReadFile(receiptPath(root))
	if err != nil {
		return r, errors.New("no hay recibo: ejecuta 'v-ia review check'")
	}
	return r, json.Unmarshal(raw, &r)
}

// Load devuelve el último recibo guardado (para mostrarlo; no lo valida).
func Load(root string) (Receipt, error) { return load(root) }

// Reset descarta un recibo (típicamente escalado) para empezar un linaje nuevo: decisión humana.
func Reset(root string) error {
	if err := os.Remove(receiptPath(root)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

var severe = map[string]bool{"BLOCKER": true, "CRITICAL": true}
var causal = map[string]bool{"introduced": true, "behavior-activated": true, "worsened": true}
var nonCausal = map[string]bool{"pre-existing": true, "base-only": true, "unknown": true}

func isBlocking(f Finding) bool { return severe[f.Severity] && causal[f.CausalDisposition] }

func validate(in ReviewInput, r Receipt) error {
	if strings.TrimSpace(in.Reviewer) == "" {
		return errors.New("falta 'reviewer'")
	}
	has := false
	for _, l := range r.SelectedLenses {
		has = has || l == in.Lens
	}
	if !has {
		return fmt.Errorf("la lente %q no está entre las seleccionadas para este recibo: %s", in.Lens, strings.Join(r.SelectedLenses, ", "))
	}
	if in.Inspection.Status != "complete" {
		return errors.New("inspection.status debe ser \"complete\": la revisión tiene que cubrir todo el cambio")
	}
	seen := map[string]bool{}
	for _, p := range in.Inspection.Paths {
		seen[p] = true
	}
	var missing []string
	for _, p := range r.Paths {
		if !seen[p] {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("inspection.paths no cubre el manifiesto; faltan %d ruta(s): %s", len(missing), strings.Join(firstN(missing, 5), ", "))
	}
	for i, f := range in.Findings {
		if strings.TrimSpace(f.Location) == "" || strings.TrimSpace(f.Claim) == "" {
			return fmt.Errorf("hallazgo %d: location y claim son obligatorios", i+1)
		}
		switch f.Severity {
		case "BLOCKER", "CRITICAL", "WARNING", "SUGGESTION":
		default:
			return fmt.Errorf("hallazgo %d: severity inválida %q", i+1, f.Severity)
		}
		if severe[f.Severity] {
			if !causal[f.CausalDisposition] && !nonCausal[f.CausalDisposition] {
				return fmt.Errorf("hallazgo %d: un severo exige causal_disposition (introduced | behavior-activated | worsened | pre-existing | base-only | unknown)", i+1)
			}
			if causal[f.CausalDisposition] && len(f.ProofRefs) == 0 {
				return fmt.Errorf("hallazgo %d: un severo causado por el cambio exige proof_refs (prueba reproducible)", i+1)
			}
		}
	}
	return nil
}

func firstN(s []string, n int) []string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// RecordReview admite la revisión de una lente. El veredicto lo calcula V-IA: "changes" si hay
// algún severo causado por el cambio con prueba; si no, "approve". Devuelve el recibo actualizado.
func RecordReview(root string, in ReviewInput) (Receipt, error) {
	r, err := load(root)
	if err != nil {
		return r, err
	}
	if r.Result != "pass" {
		return r, errors.New("el recibo no es 'pass': no se puede revisar un cambio que no pasa sus comprobaciones")
	}
	if !r.Risk.ReviewDue {
		return r, errors.New("este cambio no exige revisión independiente (riesgo " + r.Risk.Tier + ")")
	}
	if r.State == StateEscalated || r.State == StateApproved {
		return r, fmt.Errorf("el recibo ya está %s: no admite más revisiones", r.State)
	}
	cur, err := currentID(root, r.Base)
	if err != nil {
		return r, err
	}
	if cur != r.CandidateID || in.CandidateID != r.CandidateID {
		return r, errors.New("el árbol cambió (o candidate_id no coincide) desde 'review check': vuelve a ejecutarlo antes de revisar")
	}
	if err := validate(in, r); err != nil {
		return r, err
	}
	for _, x := range r.Reviews {
		if x.Lens == in.Lens {
			return r, fmt.Errorf("la lente %q ya tiene veredicto para este candidato: corrige el código y ejecuta 'v-ia review check'", in.Lens)
		}
	}
	lr := LensReview{Lens: in.Lens, Reviewer: in.Reviewer, Inspected: len(in.Inspection.Paths), Findings: in.Findings,
		Evidence: in.Evidence, Verdict: "approve", At: time.Now().UTC().Format(time.RFC3339)}
	for _, f := range in.Findings {
		if isBlocking(f) {
			lr.Verdict = "changes"
		}
	}
	r.Reviews = append(r.Reviews, lr)
	sort.Slice(r.Reviews, func(i, j int) bool { return r.Reviews[i].Lens < r.Reviews[j].Lens })
	r.State = reviewState(r)
	return r, save(root, r)
}

func reviewState(r Receipt) string {
	done := map[string]bool{}
	changes := false
	for _, x := range r.Reviews {
		done[x.Lens] = true
		changes = changes || x.Verdict == "changes"
	}
	for _, l := range r.SelectedLenses {
		if !done[l] {
			return StateChecking
		}
	}
	switch {
	case !changes:
		return StateApproved
	case r.Attempt >= 2:
		return StateEscalated // la única corrección acotada ya se usó
	}
	return StateCorrectionRequired
}

// Pending devuelve las lentes que aún no tienen revisión.
func (r Receipt) Pending() []string {
	done := map[string]bool{}
	for _, x := range r.Reviews {
		done[x.Lens] = true
	}
	var out []string
	for _, l := range r.SelectedLenses {
		if !done[l] {
			out = append(out, l)
		}
	}
	return out
}

// Verify confirma que hay un recibo pass para el árbol actual. Con requireAck (RDD activo) y revisión
// debida, exige además que todas las lentes seleccionadas hayan aprobado ese mismo candidato.
func Verify(root string, requireAck bool) error {
	r, err := load(root)
	if err != nil {
		return err
	}
	if r.Result == "stale" {
		return errors.New("el recibo es 'stale': el árbol cambió mientras corrían las comprobaciones; vuelve a ejecutar 'v-ia review check'")
	}
	if r.Result != "pass" {
		return errors.New("el último recibo es fail")
	}
	cur, err := currentID(root, r.Base)
	if err != nil {
		return err
	}
	if cur != r.CandidateID {
		return errors.New("el recibo es de otra versión: el árbol cambió desde la revisión")
	}
	if requireAck && r.Risk.ReviewDue {
		switch r.State {
		case StateApproved:
		case StateChecking:
			return fmt.Errorf("riesgo %s: faltan las revisiones independientes de: %s (subagente v-ia-reviewer → v-ia review ack)", strings.ToUpper(r.Risk.Tier), strings.Join(r.Pending(), ", "))
		case StateCorrectionRequired:
			return errors.New("un revisor encontró un bloqueante causado por el cambio: aplica la única corrección acotada y vuelve a ejecutar 'v-ia review check'")
		case StateEscalated:
			return errors.New("revisión ESCALADA: la única corrección no bastó; decide tú cómo seguir ('v-ia review reset')")
		default:
			return fmt.Errorf("estado de revisión no válido para entregar: %s", r.State)
		}
	}
	return nil
}

// StatusInfo resume el estado para el orquestador (sin inferir nada de prosa).
type StatusInfo struct {
	State          string     `json:"state"`  // working | checking | ready | needs_decision
	Detail         string     `json:"detail"` // estado interno del recibo
	NextTransition string     `json:"next_transition"`
	Pending        []string   `json:"pending_lenses,omitempty"`
	Attempt        int        `json:"attempt,omitempty"`
	Tier           string     `json:"tier,omitempty"`
	Due            bool       `json:"review_due"`
	Paths          []string   `json:"paths,omitempty"`
	CandidateID    string     `json:"candidate_id,omitempty"`
	Request        string     `json:"request,omitempty"`
	Blocking       []Blocking `json:"blocking,omitempty"`
}

// Status clasifica el recibo actual contra el árbol vivo.
func Status(root string) StatusInfo {
	r, err := load(root)
	if err != nil {
		return StatusInfo{State: PublicWorking, Detail: "sin_recibo", NextTransition: "run: v-ia review check"}
	}
	s := StatusInfo{Detail: r.State, Attempt: r.Attempt, Tier: r.Risk.Tier, Due: r.Risk.ReviewDue, Paths: r.Paths, CandidateID: r.CandidateID, Request: r.Request, Blocking: r.Blocking()}
	if cur, err := currentID(root, r.Base); err != nil || cur != r.CandidateID {
		s.State, s.Detail, s.NextTransition = PublicWorking, "candidato_cambiado", "run: v-ia review check"
		return s
	}
	switch r.State {
	case StateFailed, StateStale:
		s.State, s.NextTransition = PublicWorking, "fix: corrige y ejecuta v-ia review check"
	case StateChecking:
		s.State, s.Pending = PublicChecking, r.Pending()
		s.NextTransition = "collect: revisión independiente de las lentes " + strings.Join(s.Pending, ", ") + " (v-ia review ack)"
	case StateCorrectionRequired:
		s.State, s.NextTransition = PublicWorking, "fix: una única corrección acotada y luego v-ia review check (un falso positivo se refuta con v-ia review refute)"
	case StateEscalated:
		s.State, s.NextTransition = PublicNeedsDecision, "stop: decide el usuario (v-ia review reset)"
	default: // ready | approved
		s.State, s.NextTransition = PublicReady, "none"
	}
	return s
}

// shellCommand ejecuta c con la shell del sistema: sh en Unix, cmd en Windows.
func shellCommand(ctx context.Context, c string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/C", c)
	}
	return exec.CommandContext(ctx, "sh", "-c", c)
}
