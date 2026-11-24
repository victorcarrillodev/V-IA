// Package hook hace cumplir ODD por mecanismo: bloquea acciones que las reglas prohíben.
// Se conecta a los hooks de Claude Code (PreToolUse, UserPromptSubmit, Stop).
package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"v-ia/internal/adapters"
	"v-ia/internal/guard"
	"v-ia/internal/registry"
	"v-ia/internal/review"
	"v-ia/internal/task"
)

const (
	ModeRead     = "read"
	ModeEdit     = "edit"
	ModeTracking = "tracking"
)

// State es la clasificación vigente; vive fuera del modelo, en .v-ia/state.json.
type State struct {
	Mode       string `json:"mode"`
	DeliveryOK bool   `json:"delivery_ok"`
}

// Input es el JSON que Claude Code entrega al hook por stdin.
type Input struct {
	CWD            string `json:"cwd"`
	Event          string `json:"hook_event_name"`
	Tool           string `json:"tool_name"`
	Prompt         string `json:"prompt"`
	StopHookActive bool   `json:"stop_hook_active"`
	TurnID         string `json:"turn_id"` // solo Codex lo envía
	ToolInput      struct {
		FilePath     string `json:"file_path"`
		NotebookPath string `json:"notebook_path"`
		Command      string `json:"command"`
	} `json:"tool_input"`
}

// Result: Block => exit 2 con Msg por stderr; si no, Msg (opcional) va a stdout.
type Result struct {
	Block bool
	Msg   string
}

func statePath(root string) string { return filepath.Join(root, ".v-ia", "state.json") }

func LoadState(root string) State {
	var s State
	if b, err := os.ReadFile(statePath(root)); err == nil {
		_ = json.Unmarshal(b, &s)
	}
	return s
}

func SaveState(root string, s State) error {
	if err := os.MkdirAll(filepath.Dir(statePath(root)), 0o755); err != nil {
		return err
	}
	b, _ := json.Marshal(s)
	return os.WriteFile(statePath(root), b, 0o644)
}

var (
	noEditRe  = regexp.MustCompile(`(?i)sin (modificar|editar|tocar|cambiar)|no (modifiques|edites|toques|cambies)|solo lectura|read[- ]only`)
	readRe    = regexp.MustCompile(`(?i)\b(expl[ií]ca\w*|analiza\w*|prop[oó]n\w*|qu[eé] hace|c[oó]mo funciona)\b`)
	editRe    = regexp.MustCompile(`(?i)\b(corrige|arregla|implementa|crea|a[ñn]ade|agrega|cambia|elimina|borra|escribe|refactoriza|actualiza|modifica|edita|haz|hazlo|instala|fix|add|implement)\w*\b`)
	deliverRe = regexp.MustCompile(`(?i)\b(commit|push|pull request|release|publica|sube)\b|\bpr\b`)
	noDeliver = regexp.MustCompile(`(?i)(sin|no)\s+(hag\w+\s+|hacer\s+)?(commit|push|pr|release)`)
	dangerRe  = []*regexp.Regexp{
		regexp.MustCompile(`\brm\s+(-\S*\s+)*-\S*[rR]\S*\s+(-\S+\s+)*(/|~|\$HOME|\.|\.\.|/\*|~/\*)(\s|$|;|&|\|)`),
		regexp.MustCompile(`\bgit\s+(-\S+\s+)*push\b[^;&|\n]*(\s--force(\s|$)|\s-f(\s|$))`),
		regexp.MustCompile(`\bgit\s+(-\S+\s+)*reset\s+--hard\b`),
		regexp.MustCompile(`\bgit\s+(-\S+\s+)*clean\s+-\S*f`),
		regexp.MustCompile(`\bchmod\s+-R\s+0?777\b`),
	}
	deliveryRe = regexp.MustCompile(`\bgit\s+(-[Cc]\s+\S+\s+|-\S+\s+)*(commit|push|tag)\b|\bgh\s+(pr\s+(create|merge)|release\s+create)\b`)
)

// Classify decide el modo y si el usuario autorizó entrega, solo con la petición.
// Un "sin modificar" explícito gana siempre.
func Classify(prompt string) (mode string, delivery bool) {
	delivery = deliverRe.MatchString(prompt) && !noDeliver.MatchString(prompt)
	switch {
	case noEditRe.MatchString(prompt):
		return ModeRead, delivery
	case readRe.MatchString(prompt) && !editRe.MatchString(prompt):
		return ModeRead, delivery
	}
	return ModeEdit, delivery
}

// protectedRe: estado que gestiona v-ia y que un agente no debe reescribir (recibo, modo RDD, estado, TDD).
var protectedRe = regexp.MustCompile(`(?:^|[^\w.-])\.v-ia[/\\](receipt\.json|review-mode|state\.json|guard\.json|tdd)\b`)

func isEditTool(t string) bool {
	switch t {
	case "Edit", "Write", "MultiEdit", "NotebookEdit":
		return true
	}
	return false
}

func hasTaskFile(root string) bool {
	m, _ := filepath.Glob(filepath.Join(root, "odd", "tasks", "*.md"))
	return len(m) > 0
}

func relTo(root, p string) string {
	if filepath.IsAbs(p) {
		if r, err := filepath.Rel(root, p); err == nil {
			return r
		}
	}
	return p
}

func underTasks(root, p string) bool {
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	rel, err := filepath.Rel(filepath.Join(root, "odd", "tasks"), p)
	return err == nil && !strings.HasPrefix(rel, "..")
}

// Handle aplica las reglas ODD al evento recibido.
func Handle(in Input) Result {
	normalize(&in)
	root := in.CWD
	if root == "" {
		root, _ = os.Getwd()
	}
	st := LoadState(root)
	switch in.Event {
	case "UserPromptSubmit":
		mode, del := Classify(in.Prompt)
		if st.Mode == ModeTracking && mode == ModeEdit {
			mode = ModeTracking // el seguimiento sobrevive entre turnos hasta `v-ia mode edit`
		}
		st = State{Mode: mode, DeliveryOK: del}
		if err := SaveState(root, st); err != nil {
			return Result{Msg: "[V-IA] no pude guardar el estado: " + err.Error()}
		}
		if mode == ModeRead {
			_ = guard.Start(root)
			return Result{Msg: "[V-IA] modo LECTURA: edición bloqueada; no modifiques archivos."}
		}
		return Result{}
	case "PreToolUse":
		return preTool(root, st, in)
	case "Stop":
		return stop(root, st, in)
	case "SessionStart":
		return sessionStart(root)
	}
	return Result{}
}

// sessionStart recupera memoria (Engram, si existe), refresca el registro de skills
// si el proyecto ya lo usa y recuerda el modo vigente.
func sessionStart(root string) Result {
	var parts []string
	if exe, err := exec.LookPath("engram"); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, exe, "context")
		cmd.Dir = root
		var out bytes.Buffer
		cmd.Stdout = &out
		if cmd.Run() == nil && out.Len() > 0 {
			b := out.String()
			if len(b) > 3000 {
				b = b[:3000] + "\n…(recortado)"
			}
			parts = append(parts, "[V-IA] Memoria del proyecto (contrástala con el código; si discrepan gana el código):\n"+b)
		}
	}
	if _, err := os.Stat(registry.Path(root)); err == nil {
		if home, err := adapters.Home(); err == nil {
			_, _ = registry.Refresh(home, root)
		}
		parts = append(parts, "[V-IA] Lee .atl/skill-registry.md: lista las skills disponibles.")
	}
	return Result{Msg: strings.Join(parts, "\n")}
}

func preTool(root string, st State, in Input) Result {
	switch {
	case isEditTool(in.Tool):
		p := in.ToolInput.FilePath
		if p == "" {
			p = in.ToolInput.NotebookPath
		}
		if protectedRe.MatchString(filepath.ToSlash(p)) {
			return Result{Block: true, Msg: "V-IA/seguridad: el estado de revisión y de modo (.v-ia/receipt.json, review-mode, state.json…) lo gestiona v-ia; no lo edites a mano. Usa los comandos `v-ia review|mode|tdd`."}
		}
		switch st.Mode {
		case ModeRead:
			return Result{Block: true, Msg: "V-IA/ODD: modo lectura, la petición no autoriza editar. Responde sin modificar archivos."}
		case ModeTracking:
			if !hasTaskFile(root) && !underTasks(root, p) {
				return Result{Block: true, Msg: "V-IA/ODD: trabajo de seguimiento sin ficha. Crea odd/tasks/<feature>.md (v-ia task new <feature>) antes de editar."}
			}
		}
		if msg := tddGate(root, relTo(root, p)); msg != "" {
			return Result{Block: true, Msg: msg}
		}
	case in.Tool == "Bash":
		cmd := in.ToolInput.Command
		if protectedRe.MatchString(cmd) && writesFiles(cmd) && !isVIACommand(cmd) {
			return Result{Block: true, Msg: "V-IA/seguridad: el comando modifica el estado que gestiona v-ia (.v-ia/receipt.json, review-mode, state.json…). Usa los comandos `v-ia review|mode|tdd`."}
		}
		if st.Mode == ModeRead && writesFiles(cmd) {
			return Result{Block: true, Msg: "V-IA/ODD: modo lectura, el comando modifica archivos (redirección, sed -i, tee, mv, rm, git apply…). Responde sin modificar nada."}
		}
		for _, re := range dangerRe {
			if re.MatchString(cmd) {
				return Result{Block: true, Msg: "V-IA/seguridad: comando destructivo bloqueado (rm -rf de raíz/HOME/., push --force, reset --hard, clean -f, chmod -R 777). Pídele al usuario que lo ejecute él mismo."}
			}
		}
		if deliveryRe.MatchString(cmd) {
			if !st.DeliveryOK {
				return Result{Block: true, Msg: "V-IA/ODD: commit/push/PR/release requieren que el usuario lo pida de forma explícita en su mensaje."}
			}
			if msg := receiptGate(root); msg != "" {
				return Result{Block: true, Msg: msg}
			}
		}
	}
	return Result{}
}

// receiptGate exige recibo válido para entregar cuando RDD está activo y el proyecto
// tiene comprobaciones definidas. Sin comprobaciones no hay nada que exigir.
func receiptGate(root string) string {
	home, err := adapters.Home()
	if err != nil {
		return ""
	}
	on, _ := review.Mode(root, home)
	if !on || len(review.DetectCommands(root)) == 0 {
		return ""
	}
	if err := review.Verify(root, true); err != nil {
		return "V-IA/RDD: no se puede entregar sin recibo válido: " + err.Error() + ". Ejecuta `v-ia review check`."
	}
	return ""
}

func stop(root string, st State, in Input) Result {
	if in.StopHookActive { // evita bucles: solo se bloquea una vez por turno
		return Result{}
	}
	var msgs []string
	if st.Mode == ModeRead {
		if ch, err := guard.Diff(root); err == nil && len(ch) > 0 {
			msgs = append(msgs, "Modo lectura: se detectaron cambios en archivos. Revierte estos cambios:\n  "+strings.Join(ch, "\n  "))
		}
	}
	if res, _, err := task.CheckAll(root); err == nil {
		for f, ps := range res {
			for _, p := range ps {
				msgs = append(msgs, fmt.Sprintf("%s: %s", f, p))
			}
		}
	}
	if len(msgs) == 0 {
		if m := reviewReminder(root); m != "" {
			return Result{Block: true, Msg: m}
		}
		return Result{}
	}
	return Result{Block: true, Msg: "V-IA/ODD: no cierres todavía.\n" + strings.Join(msgs, "\n")}
}

// isVIACommand indica si el comando es una invocación directa de v-ia (que sí puede tocar su estado).
func isVIACommand(cmd string) bool {
	f := strings.Fields(strings.TrimSpace(cmd))
	return len(f) > 0 && (f[0] == "v-ia" || strings.HasSuffix(f[0], "/v-ia")) && !strings.ContainsAny(cmd, ";&|`$(")
}
