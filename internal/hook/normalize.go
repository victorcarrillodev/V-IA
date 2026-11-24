package hook

import "strings"

// Category resume un bloqueo en una etiqueta estable (para el recuento local).
func Category(msg string) string {
	switch {
	case strings.Contains(msg, "TDD"):
		return "tdd"
	case strings.Contains(msg, "modo lectura"):
		return "read-mode"
	case strings.Contains(msg, "destructivo"):
		return "destructive"
	case strings.Contains(msg, "seguimiento sin ficha"):
		return "tracking-no-task"
	case strings.Contains(msg, "RDD"):
		return "receipt"
	case strings.Contains(msg, "commit/push"):
		return "delivery"
	}
	return "other"
}

// normalize traduce los nombres de eventos y herramientas de otros clientes (Gemini CLI)
// a los de Claude Code, que son los que entiende Handle.
func normalize(in *Input) {
	switch in.Event {
	case "BeforeTool":
		in.Event = "PreToolUse"
	case "BeforeAgent", "beforeSubmitPrompt":
		in.Event = "UserPromptSubmit"
	case "preToolUse": // Cursor
		in.Event = "PreToolUse"
	case "SessionStart", "SessionEnd", "UserPromptSubmit", "PreToolUse", "Stop":
	}
	switch strings.ToLower(in.Tool) {
	case "write_file", "replace", "edit_file", "edit", "write", "multiedit", "apply_patch", "patch", "delete":
		if !isEditTool(in.Tool) {
			in.Tool = "Edit"
		}
	case "run_shell_command", "shell", "bash":
		in.Tool = "Bash"
	}
}
