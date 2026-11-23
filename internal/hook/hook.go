// Package hook hace cumplir ODD por mecanismo: bloquea acciones que las reglas prohíben.
// Se conecta a los hooks de Claude Code (PreToolUse, UserPromptSubmit, Stop).
package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"v-ia/internal/guard"
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
	noEditRe   = regexp.MustCompile(`(?i)sin (modificar|editar|tocar|cambiar)|no (modifiques|edites|toques|cambies)|solo lectura|read[- ]only`)
	readRe     = regexp.MustCompile(`(?i)\b(expl[ií]ca\w*|analiza\w*|prop[oó]n\w*|qu[eé] hace|c[oó]mo funciona)\b`)
	editRe     = regexp.MustCompile(`(?i)\b(corrige|arregla|implementa|crea|a[ñn]ade|agrega|cambia|elimina|borra|escribe|refactoriza|actualiza|modifica|edita|haz|hazlo|instala|fix|add|implement)\w*\b`)
	deliverRe  = regexp.MustCompile(`(?i)\b(commit|push|pull request|release|publica|sube)\b|\bpr\b`)
	noDeliver  = regexp.MustCompile(`(?i)(sin|no)\s+(hag\w+\s+|hacer\s+)?(commit|push|pr|release)`)
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

func underTasks(root, p string) bool {
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	rel, err := filepath.Rel(filepath.Join(root, "odd", "tasks"), p)
	return err == nil && !strings.HasPrefix(rel, "..")
}

// Handle aplica las reglas ODD al evento recibido.
func Handle(in Input) Result {
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
	}
	return Result{}
}

func preTool(root string, st State, in Input) Result {
	switch {
	case isEditTool(in.Tool):
		p := in.ToolInput.FilePath
		if p == "" {
			p = in.ToolInput.NotebookPath
		}
		switch st.Mode {
		case ModeRead:
			return Result{Block: true, Msg: "V-IA/ODD: modo lectura, la petición no autoriza editar. Responde sin modificar archivos."}
		case ModeTracking:
			if !hasTaskFile(root) && !underTasks(root, p) {
				return Result{Block: true, Msg: "V-IA/ODD: trabajo de seguimiento sin ficha. Crea odd/tasks/<feature>.md (v-ia task new <feature>) antes de editar."}
			}
		}
	case in.Tool == "Bash" && deliveryRe.MatchString(in.ToolInput.Command):
		if !st.DeliveryOK {
			return Result{Block: true, Msg: "V-IA/ODD: commit/push/PR/release requieren que el usuario lo pida de forma explícita en su mensaje."}
		}
	}
	return Result{}
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
		return Result{}
	}
	return Result{Block: true, Msg: "V-IA/ODD: no cierres todavía.\n" + strings.Join(msgs, "\n")}
}
