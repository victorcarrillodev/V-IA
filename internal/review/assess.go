package review

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	TierPassive = "passive"
	TierMedium  = "medium"
	TierHigh    = "high"
	TierUnknown = "unknown" // una evaluación fallida nunca implica riesgo bajo: se trata como alto
)

// Motivos por los que un cambio exige (o no) revisión independiente.
const (
	DueHighRisk         = "high_risk"
	DueAssessmentFailed = "assessment_failed"
	DueUnderBudget      = "under_budget"
	DuePassive          = "passive"
)

// Assessment es la evaluación de riesgo del cambio (lectura, no escribe nada).
// El tier lo deciden las señales, nunca el número de archivos ni el modelo.
type Assessment struct {
	Tier            string      `json:"tier"`
	Reasons         []string    `json:"reasons"`
	Signals         []string    `json:"signals,omitempty"`
	Files           int         `json:"files"`
	Lines           int         `json:"lines"`
	Paths           []string    `json:"paths,omitempty"`
	ReviewDue       bool        `json:"review_due"`
	ReviewDueReason string      `json:"review_due_reason"`
	Escalation      *Escalation `json:"agent_escalation,omitempty"`
}

// Escalation es la subida de riesgo que declara el agente: nunca baja un tier.
type Escalation struct {
	Item   int    `json:"item"`
	Reason string `json:"reason"`
}

// HighRiskItems son los seis ítems cerrados de riesgo alto.
var HighRiskItems = map[int]string{
	1: "datos o efectos irreversibles que no se pueden reconstruir",
	2: "seguridad",
	3: "contratos que otros consumen (API, flags de CLI, formatos de configuración)",
	4: "concurrencia",
	5: "entrega o entorno (instaladores, release, CI, dependencias)",
	6: "ningún test detectaría una regresión",
}

var (
	// documentación pasiva: se decide por extensión, nunca por el directorio (docs/deploy.sh no es doc)
	docRe  = regexp.MustCompile(`(?i)\.(md|txt|rst|adoc|png|jpe?g|gif|svg)$|(^|/)(LICENSE|COPYING|NOTICE)$`)
	testRe = regexp.MustCompile(`(?i)_test\.go$|\.(test|spec)\.[a-z]+$|(^|/)(tests?|__tests__|testdata|fixtures?)/|(^|/)test_[^/]+\.py$|_test\.py$`)
	// extensiones de código ejecutable o de configuración operativa
	codeRe     = regexp.MustCompile(`(?i)\.(go|py|js|jsx|ts|tsx|mjs|cjs|rs|java|kt|rb|php|c|h|cc|cpp|cs|swift|sh|bash|zsh|sql|ps1)$|(^|/)(Dockerfile|Makefile)[^/]*$|(^|/)\.github/workflows/[^/]+\.ya?ml$`)
	highPathRe = regexp.MustCompile(`(?i)(^|/)(auth\w*|security|crypto\w*|secrets?|tokens?|passw\w*|permissions?|migrations?|schema\w*|payments?|webhooks?)(/|\.|_|-|$)|\.sql$|(^|/)\.github/workflows/|(^|/)Dockerfile|(^|/)(release|deploy)\w*(/|\.|$)`)
	depRe      = regexp.MustCompile(`(^|/)(go\.mod|go\.sum|package(-lock)?\.json|pnpm-lock\.yaml|yarn\.lock|Cargo\.(toml|lock)|requirements[\w.-]*\.txt|pyproject\.toml)$`)
	configRe   = regexp.MustCompile(`(?i)(^|/)\.env[\w.]*$|\.(json|ya?ml|toml|ini)$`)
	highTextRe = regexp.MustCompile(`(?i)\b(DROP\s+(TABLE|DATABASE)|DELETE\s+FROM|ALTER\s+TABLE|TRUNCATE)\b|(api[_-]?key|secret|passw(or)?d|private[_-]?key)\s*[:=]\s*["'][^"']{6,}|sync\.(Mutex|RWMutex|WaitGroup)|\bgo\s+func\b|atomic\.|chmod\s+-R|exec\.Command|child_process|subprocess\.|os\.system\(|InsecureSkipVerify|verify\s*=\s*False|\beval\(|pickle\.loads|rejectUnauthorized\s*:\s*false`)
)

func git(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if i := strings.IndexByte(msg, '\n'); i >= 0 {
			msg = msg[:i] // solo la primera línea: git vuelca su ayuda entera en algunos errores
		}
		return "", fmt.Errorf("git %s: %v %s", strings.Join(args, " "), err, msg)
	}
	return out.String(), nil
}

func unknown(why string) Assessment {
	return Assessment{Tier: TierUnknown, Reasons: []string{why}, ReviewDue: true, ReviewDueReason: DueAssessmentFailed}
}

// Assess congela el cambio actual respecto a base (por defecto HEAD) y lo clasifica.
func Assess(root, base string) Assessment {
	c, err := TakeCandidate(root, base)
	if err != nil {
		return unknown("no se pudo evaluar el cambio: " + err.Error())
	}
	return AssessCandidate(root, c)
}

// AssessCandidate clasifica un candidato ya congelado (las líneas de archivos nuevos son reales).
func AssessCandidate(root string, c Candidate) Assessment {
	num, err := git(root, "diff-tree", "-r", "--numstat", "--no-renames", c.BaseTree, c.CandidateTree)
	if err != nil {
		return unknown("no se pudo evaluar: " + err.Error())
	}
	files := map[string]int{}
	for _, l := range strings.Split(strings.TrimSpace(num), "\n") {
		f := strings.SplitN(l, "\t", 3)
		if len(f) != 3 {
			continue
		}
		a, _ := strconv.Atoi(f[0]) // binarios: "-" => 0
		d, _ := strconv.Atoi(f[1])
		files[f[2]] = a + d
	}
	patch, err := git(root, "diff-tree", "-r", "-p", "-U0", "--no-renames", c.BaseTree, c.CandidateTree)
	if err != nil {
		return unknown("no se pudo evaluar: " + err.Error())
	}
	a := classify(files, addedLines(patch))
	a.Paths = c.Paths
	return a
}

// addedLines devuelve, por archivo, las líneas añadidas de un parche -U0.
func addedLines(patch string) map[string][]string {
	out := map[string][]string{}
	cur := ""
	for _, l := range strings.Split(patch, "\n") {
		switch {
		case strings.HasPrefix(l, "+++ b/"):
			cur = strings.TrimPrefix(l, "+++ b/")
		case strings.HasPrefix(l, "+++ "), strings.HasPrefix(l, "--- "):
			if strings.HasPrefix(l, "+++ /dev/null") {
				cur = ""
			}
		case cur != "" && strings.HasPrefix(l, "+"):
			out[cur] = append(out[cur], l[1:])
		}
	}
	return out
}

func classify(files map[string]int, added map[string][]string) Assessment {
	a := Assessment{Files: len(files)}
	src, tests := 0, 0
	sig := map[string]bool{}
	cfg := false
	names := make([]string, 0, len(files))
	for f := range files {
		names = append(names, f)
	}
	sort.Strings(names)
	for _, f := range names {
		a.Lines += files[f]
		isTest, isDoc := testRe.MatchString(f), docRe.MatchString(f)
		switch {
		case isTest:
			tests++
		case isDoc:
		default:
			src++
		}
		if isTest || isDoc {
			continue // los nombres de ruta en tests y documentación no son evidencia de riesgo
		}
		if codeRe.MatchString(f) && highPathRe.MatchString(f) {
			sig["path"] = true
			a.Reasons = append(a.Reasons, "ruta sensible: "+f)
		}
		if depRe.MatchString(f) || (configRe.MatchString(f) && !codeRe.MatchString(f)) {
			cfg = true
		}
		if codeRe.MatchString(f) {
			for _, l := range added[f] {
				if highTextRe.MatchString(l) {
					sig["text"] = true
					a.Reasons = append(a.Reasons, "contenido sensible (datos/secretos/concurrencia/exec) en "+f+": "+strings.TrimSpace(trunc(l, 60)))
					break
				}
			}
		}
	}
	for s := range sig {
		a.Signals = append(a.Signals, s)
	}
	sort.Strings(a.Signals)
	switch {
	case len(files) == 0:
		a.Tier = TierPassive
		a.Reasons = append(a.Reasons, "sin cambios")
	case len(sig) > 0:
		a.Tier = TierHigh
	case src == 0:
		a.Tier = TierPassive
		a.Reasons = append(a.Reasons, "solo docs/tests")
	case cfg:
		a.Tier = TierMedium
		a.Signals = append(a.Signals, "config")
		a.Reasons = append(a.Reasons, "cambio de dependencias o configuración")
	case a.Lines >= 150 || src >= 5:
		a.Tier = TierMedium
		a.Reasons = append(a.Reasons, fmt.Sprintf("cambio amplio (%d líneas, %d archivos de código)", a.Lines, src))
	case tests == 0 && a.Lines > 30:
		a.Tier = TierMedium
		a.Reasons = append(a.Reasons, "código sin tests en el mismo cambio")
	default:
		a.Tier = TierPassive
		a.Reasons = append(a.Reasons, "cambio pequeño y acotado")
	}
	a.setDue()
	return a
}

func (a *Assessment) setDue() {
	switch a.Tier {
	case TierHigh:
		a.ReviewDue, a.ReviewDueReason = true, DueHighRisk
	case TierUnknown:
		a.ReviewDue, a.ReviewDueReason = true, DueAssessmentFailed
	case TierMedium:
		a.ReviewDue, a.ReviewDueReason = false, DueUnderBudget
	default:
		a.ReviewDue, a.ReviewDueReason = false, DuePassive
	}
}

// Escalate sube a alto el riesgo por un ítem cerrado (1-6) que el agente nombra con su razón.
// Nunca baja un tier. La razón (una línea, ≤500 bytes) queda congelada en el recibo.
func (a *Assessment) Escalate(item int, reason string) error {
	if _, ok := HighRiskItems[item]; !ok {
		return fmt.Errorf("ítem de riesgo inválido %d (1-6)", item)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || strings.ContainsAny(reason, "\r\n") || len(reason) > 500 {
		return errors.New("la razón del escalado debe ser una sola línea de 1 a 500 bytes")
	}
	a.Escalation = &Escalation{Item: item, Reason: reason}
	if a.Tier != TierHigh {
		a.Tier = TierHigh
		a.Reasons = append(a.Reasons, fmt.Sprintf("escalado por el agente: ítem %d (%s): %s", item, HighRiskItems[item], reason))
		a.Signals = append(a.Signals, "agent_escalation")
	}
	a.setDue()
	return nil
}

// AllLenses es el orden canónico 4R.
var AllLenses = []string{"risk", "resilience", "readability", "reliability"}

// DefaultLenses: passive 0, medium 1 (reliability), high las 4.
func DefaultLenses(tier string) []string {
	switch tier {
	case TierHigh, TierUnknown:
		return append([]string(nil), AllLenses...)
	case TierMedium:
		return []string{"reliability"}
	}
	return nil
}

// SelectLenses valida la elección del agente (solo en medium/high), con razón obligatoria, y la
// devuelve en el orden canónico 4R.
func SelectLenses(tier string, custom []string, reason string) ([]string, error) {
	if len(custom) == 0 {
		return DefaultLenses(tier), nil
	}
	if tier == TierPassive {
		return nil, errors.New("un cambio pasivo no tiene lentes que seleccionar")
	}
	if strings.TrimSpace(reason) == "" {
		return nil, errors.New("--lenses exige --lenses-reason")
	}
	want := map[string]bool{}
	for _, l := range custom {
		ok := false
		for _, k := range AllLenses {
			ok = ok || k == l
		}
		if !ok {
			return nil, fmt.Errorf("lente desconocida %q (risk, resilience, readability, reliability)", l)
		}
		if want[l] {
			return nil, fmt.Errorf("lente repetida %q", l)
		}
		want[l] = true
	}
	var out []string
	for _, k := range AllLenses {
		if want[k] {
			out = append(out, k)
		}
	}
	return out, nil
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
