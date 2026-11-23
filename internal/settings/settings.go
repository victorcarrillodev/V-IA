// Package settings gestiona los hooks de V-IA dentro de ~/.claude/settings.json
// sin tocar el resto de la configuración del usuario.
package settings

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var managedRe = regexp.MustCompile(`v-ia"? hook (pre|prompt|stop)$`)

type spec struct {
	event, matcher, arg string
}

var specs = []spec{
	{"PreToolUse", "Edit|Write|MultiEdit|NotebookEdit|Bash", "pre"},
	{"UserPromptSubmit", "", "prompt"},
	{"Stop", "", "stop"},
}

func command(exe, arg string) string {
	if strings.ContainsAny(exe, " \t") {
		exe = `"` + exe + `"`
	}
	return exe + " hook " + arg
}

func parse(existing string) (map[string]any, error) {
	m := map[string]any{}
	if strings.TrimSpace(existing) == "" {
		return m, nil
	}
	if err := json.Unmarshal([]byte(existing), &m); err != nil {
		return nil, fmt.Errorf("settings.json no es JSON válido (no se toca): %w", err)
	}
	return m, nil
}

func isManaged(h any) bool {
	hm, ok := h.(map[string]any)
	if !ok {
		return false
	}
	c, _ := hm["command"].(string)
	return managedRe.MatchString(c)
}

// strip quita nuestros hooks y deja los del usuario.
func strip(m map[string]any) {
	hooks, _ := m["hooks"].(map[string]any)
	for ev, v := range hooks {
		groups, _ := v.([]any)
		var keep []any
		for _, g := range groups {
			gm, ok := g.(map[string]any)
			if !ok {
				keep = append(keep, g)
				continue
			}
			inner, _ := gm["hooks"].([]any)
			var ih []any
			for _, h := range inner {
				if !isManaged(h) {
					ih = append(ih, h)
				}
			}
			if len(ih) == 0 && len(inner) > 0 {
				continue
			}
			gm["hooks"] = ih
			keep = append(keep, gm)
		}
		if len(keep) == 0 {
			delete(hooks, ev)
		} else {
			hooks[ev] = keep
		}
	}
	if len(hooks) == 0 {
		delete(m, "hooks")
	}
}

func render(m map[string]any) (string, error) {
	b, err := json.MarshalIndent(m, "", "  ")
	return string(b) + "\n", err
}

// Merge añade (o actualiza) los hooks de V-IA. Es idempotente.
func Merge(existing, exe string) (string, error) {
	m, err := parse(existing)
	if err != nil {
		return "", err
	}
	strip(m)
	hooks, _ := m["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
		m["hooks"] = hooks
	}
	for _, s := range specs {
		g := map[string]any{"hooks": []any{map[string]any{"type": "command", "command": command(exe, s.arg)}}}
		if s.matcher != "" {
			g["matcher"] = s.matcher
		}
		list, _ := hooks[s.event].([]any)
		hooks[s.event] = append(list, g)
	}
	return render(m)
}

// Strip retira los hooks de V-IA.
func Strip(existing string) (string, error) {
	m, err := parse(existing)
	if err != nil {
		return "", err
	}
	strip(m)
	return render(m)
}

// Installed indica si los tres hooks de V-IA están presentes.
func Installed(existing string) bool {
	m, err := parse(existing)
	if err != nil {
		return false
	}
	hooks, _ := m["hooks"].(map[string]any)
	n := 0
	for _, s := range specs {
		groups, _ := hooks[s.event].([]any)
		for _, g := range groups {
			gm, _ := g.(map[string]any)
			inner, _ := gm["hooks"].([]any)
			for _, h := range inner {
				if isManaged(h) {
					n++
				}
			}
		}
	}
	return n >= len(specs)
}
