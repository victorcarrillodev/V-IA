package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Hooks de otros agentes (Gemini CLI, Codex, Cursor): mismo contrato que Claude Code —el comando
// recibe un JSON por stdin y el código de salida 2 bloquea la acción—, pero cada uno guarda la
// configuración con su propia forma. HookSpec describe esa forma; las entradas de V-IA se
// reconocen por su comando (`<…>/v-ia hook`), así que nunca se tocan las del usuario.

// HookEvent es un evento a enganchar y, si el agente lo admite, su filtro (regex) de herramienta.
type HookEvent struct{ Name, Matcher string }

// HookSpec describe el formato de hooks de un agente.
type HookSpec struct {
	Nested  bool // {matcher, hooks:[{type,command}]} (Gemini, Codex) o plano {command, matcher} (Cursor)
	Version int  // "version" de nivel superior (Cursor = 1); 0 = no se escribe
	Timeout int  // timeout de la entrada, en la unidad del agente (0 = omitir)
	Events  []HookEvent
}

var Specs = map[string]HookSpec{
	// SIN VERIFICAR con el agente real: formas tomadas de la documentación oficial de cada uno.
	"gemini-cli": {Nested: true, Timeout: 15000, Events: []HookEvent{{"BeforeTool", "write_file|replace|run_shell_command"}, {"BeforeAgent", ""}}},
	"codex":      {Nested: true, Timeout: 30, Events: []HookEvent{{"PreToolUse", "Bash|apply_patch|Edit|Write"}, {"UserPromptSubmit", ""}}},
	"cursor":     {Version: 1, Timeout: 15, Events: []HookEvent{{"preToolUse", "Shell|Write|Delete"}, {"beforeSubmitPrompt", ""}}},
}

func ours(cmd string) bool {
	return cmd == "v-ia hook" || strings.HasSuffix(cmd, "/v-ia hook") || strings.HasSuffix(cmd, `/v-ia" hook`)
}

// hookCommand entrecomilla la ruta del binario si lleva espacios (el agente la ejecuta con shell).
func hookCommand(exe string) string {
	if strings.ContainsAny(exe, " \t") {
		return `"` + exe + `" hook`
	}
	return exe + " hook"
}

func cmdOf(v any) []string {
	m, _ := v.(map[string]any)
	var out []string
	if c, ok := m["command"].(string); ok {
		out = append(out, c)
	}
	hs, _ := m["hooks"].([]any)
	for _, h := range hs {
		if hm, ok := h.(map[string]any); ok {
			if c, ok := hm["command"].(string); ok {
				out = append(out, c)
			}
		}
	}
	return out
}

func isOurs(v any) bool {
	for _, c := range cmdOf(v) {
		if ours(c) {
			return true
		}
	}
	return false
}

func entry(sp HookSpec, exe string, ev HookEvent) map[string]any {
	cmd := hookCommand(exe)
	h := map[string]any{"command": cmd}
	if sp.Timeout > 0 {
		h["timeout"] = sp.Timeout
	}
	if !sp.Nested {
		if ev.Matcher != "" {
			h["matcher"] = ev.Matcher
		}
		return h
	}
	h["type"] = "command"
	e := map[string]any{"hooks": []any{h}}
	if ev.Matcher != "" {
		e["matcher"] = ev.Matcher
	}
	return e
}

func parseJSON(old string) (map[string]any, error) {
	cfg := map[string]any{}
	if len(bytes.TrimSpace([]byte(old))) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(old), &cfg); err != nil {
		return nil, fmt.Errorf("la configuración de hooks no es JSON válido (no la toco): %w", err)
	}
	return cfg, nil
}

func dropOurs(list []any) []any {
	var keep []any
	for _, v := range list {
		if !isOurs(v) {
			keep = append(keep, v)
		}
	}
	return keep
}

func renderCfg(cfg map[string]any) (string, error) {
	b, err := json.MarshalIndent(cfg, "", "  ")
	return string(b) + "\n", err
}

// HooksMerge añade (o refresca) los hooks de V-IA conservando todo lo demás.
func HooksMerge(old, exe string, sp HookSpec) (string, error) {
	cfg, err := parseJSON(old)
	if err != nil {
		return "", err
	}
	if sp.Version > 0 {
		if _, ok := cfg["version"]; !ok {
			cfg["version"] = sp.Version
		}
	}
	hooks, _ := cfg["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	for _, ev := range sp.Events {
		list, _ := hooks[ev.Name].([]any)
		hooks[ev.Name] = append(dropOurs(list), entry(sp, exe, ev))
	}
	cfg["hooks"] = hooks
	return renderCfg(cfg)
}

// HooksInstalled indica si hay hooks de V-IA en la configuración.
func HooksInstalled(old string) bool {
	cfg, err := parseJSON(old)
	if err != nil {
		return false
	}
	hooks, _ := cfg["hooks"].(map[string]any)
	for _, v := range hooks {
		list, _ := v.([]any)
		for _, e := range list {
			if isOurs(e) {
				return true
			}
		}
	}
	return false
}

// HooksStrip retira solo los hooks de V-IA.
func HooksStrip(old string) (string, error) {
	cfg, err := parseJSON(old)
	if err != nil {
		return "", err
	}
	hooks, _ := cfg["hooks"].(map[string]any)
	for ev, v := range hooks {
		list, _ := v.([]any)
		if k := dropOurs(list); len(k) > 0 {
			hooks[ev] = k
		} else {
			delete(hooks, ev)
		}
	}
	if len(hooks) == 0 {
		delete(cfg, "hooks")
	}
	return renderCfg(cfg)
}
