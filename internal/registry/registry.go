// Package registry genera el índice de skills (.atl/skill-registry.md) desde disco.
package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Skill struct {
	Name, Description, Path, Scope string
}

// Path es el archivo del registro dentro de un proyecto.
func Path(root string) string { return filepath.Join(root, ".atl", "skill-registry.md") }

func parse(path string) (name, desc string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	lines := strings.Split(string(b), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", ""
	}
	for _, l := range lines[1:] {
		if strings.TrimSpace(l) == "---" {
			break
		}
		if k, v, ok := strings.Cut(l, ":"); ok {
			v = strings.Trim(strings.TrimSpace(v), `"'`)
			switch strings.TrimSpace(k) {
			case "name":
				name = v
			case "description":
				desc = v
			}
		}
	}
	return
}

func scan(dir, scope string) []Skill {
	ents, _ := os.ReadDir(dir)
	var out []Skill
	for _, e := range ents {
		p := filepath.Join(dir, e.Name(), "SKILL.md")
		name, desc := parse(p)
		if name == "" {
			continue
		}
		out = append(out, Skill{Name: name, Description: desc, Path: p, Scope: scope})
	}
	return out
}

// Scan reúne las skills; ante un nombre repetido gana la del proyecto sobre la global.
func Scan(home, root string) []Skill {
	by := map[string]Skill{}
	for _, s := range scan(filepath.Join(home, ".claude", "skills"), "global") {
		by[s.Name] = s
	}
	for _, s := range scan(filepath.Join(root, ".claude", "skills"), "proyecto") {
		by[s.Name] = s
	}
	var out []Skill
	for _, s := range by {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Render produce el índice de forma determinista (sin fechas: mismo contenido, mismos bytes).
func Render(skills []Skill) string {
	var sb strings.Builder
	sb.WriteString("# Skill registry\n\nGenerado por `v-ia skill-registry refresh`. No lo edites a mano. Prioridad: proyecto > global.\n\n")
	sb.WriteString("| Skill | Alcance | Descripción | Ruta |\n|---|---|---|---|\n")
	for _, s := range skills {
		fmt.Fprintf(&sb, "| %s | %s | %s | %s |\n", s.Name, s.Scope, strings.ReplaceAll(s.Description, "|", "/"), s.Path)
	}
	return sb.String()
}

// Refresh reescanea y escribe solo si el contenido cambia. Devuelve si escribió.
func Refresh(home, root string) (bool, error) {
	content := Render(Scan(home, root))
	if old, err := os.ReadFile(Path(root)); err == nil && string(old) == content {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(Path(root)), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(Path(root), []byte(content), 0o644)
}
