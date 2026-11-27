// Package settings gestiona los hooks de V-IA dentro de ~/.claude/settings.json
// sin tocar el resto de la configuración del usuario.
package settings

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var managedRe = regexp.MustCompile(`v-ia"? hook (pre|prompt|stop|session)$`)

type spec struct {
	event, matcher, arg string
}

var specs = []spec{
	{"SessionStart", "", "session"},
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

// sensitive son las rutas que se niegan por defecto, tanto para leer como para editar.
var sensitive = []string{
	"~/.ssh/**", "**/.env", "**/.env.*", "**/*.pem", "**/*.key",
	"**/*.p12", "**/*.pfx", "~/.aws/credentials", "~/.config/gh/hosts.yml",
	"**/secrets/**", "~/.credentials/**",
}

func rules(tool string) []string {
	out := make([]string, len(sensitive))
	for i, p := range sensitive {
		out[i] = tool + "(" + p + ")"
	}
	return out
}

// DenyList es la deny-list por defecto. Las reglas Edit cubren también Write.
var DenyList = append(rules("Read"), rules("Edit")...)

// isOldDefault dice si la lista es exactamente la que sembraban las versiones anteriores (solo
// lectura), sin cambios del usuario.
func isOldDefault(v any) bool {
	list, _ := v.([]any)
	have := map[any]bool{}
	for _, x := range list {
		if s, ok := x.(string); ok {
			have[s] = true
		}
	}
	old := rules("Read")
	if len(list) != len(old) {
		return false
	}
	for _, r := range old {
		if !have[r] {
			return false
		}
	}
	return true
}

// seedDeny crea permissions.deny si la clave no existe, o completa la lista antigua sin modificar:
// lo que el usuario ya decidió (incluida una lista vacía) no se toca.
func seedDeny(m map[string]any) {
	perms, _ := m["permissions"].(map[string]any)
	if perms == nil {
		perms = map[string]any{}
	}
	if v, ok := perms["deny"]; ok && !isOldDefault(v) {
		return
	}
	l := make([]any, len(DenyList))
	for i, d := range DenyList {
		l[i] = d
	}
	perms["deny"] = l
	m["permissions"] = perms
}

// Merge añade (o actualiza) los hooks de V-IA (si hooksOn) y, con perms, la deny-list. Es idempotente.
func Merge(existing, exe string, hooksOn, perms bool) (string, error) {
	m, err := parse(existing)
	if err != nil {
		return "", err
	}
	if perms {
		seedDeny(m)
	}
	if !hooksOn {
		return render(m) // sin hooks: solo permisos; no se tocan los hooks existentes
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
