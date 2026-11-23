// Package task crea y valida odd/tasks/*.md.
package task

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var (
	idRe      = regexp.MustCompile(`\b(T-\d+)\b`)
	checkedRe = regexp.MustCompile(`^\s*- \[[xX]\]\s*(.*)$`)
)

// Slug convierte un nombre de feature en nombre de archivo.
func Slug(s string) string {
	var sb strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
			dash = false
		} else if !dash && sb.Len() > 0 {
			sb.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(sb.String(), "-")
}

// New crea odd/tasks/<slug>.md desde la plantilla. No sobrescribe.
func New(root, feature, tmpl string, now time.Time) (string, error) {
	slug := Slug(feature)
	if slug == "" {
		return "", fmt.Errorf("nombre de feature vacío")
	}
	path := filepath.Join(root, "odd", "tasks", slug+".md")
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("%s ya existe", path)
	}
	d := now.Format("2006-01-02")
	r := strings.NewReplacer("<nombre>", feature, "<fecha>", d)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	return path, os.WriteFile(path, []byte(r.Replace(tmpl)), 0o644)
}

// Problem es un fallo de validación con la corrección sugerida.
type Problem struct {
	Line int
	Msg  string
}

func (p Problem) String() string { return fmt.Sprintf("línea %d: %s", p.Line, p.Msg) }

func section(lines []string, title string) (from, to int) {
	from, to = -1, len(lines)
	for i, l := range lines {
		if strings.HasPrefix(l, "## ") {
			if from >= 0 {
				return from, i
			}
			if strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(l, "## ")), title) {
				from = i + 1
			}
		}
	}
	return from, to
}

// Check valida: todo `[x]` de Tareas necesita fila en Evidencia con comando y resultado.
func Check(content string) []Problem {
	lines := strings.Split(content, "\n")
	var ps []Problem
	if !strings.HasPrefix(content, "---") {
		ps = append(ps, Problem{1, "falta el frontmatter (--- feature/status ... ---)"})
	}
	ev := map[string]bool{}
	if f, t := section(lines, "Evidencia"); f >= 0 {
		for _, l := range lines[f:t] {
			if !strings.HasPrefix(strings.TrimSpace(l), "|") {
				continue
			}
			cells := strings.Split(strings.Trim(strings.TrimSpace(l), "|"), "|")
			if len(cells) < 3 {
				continue
			}
			id := idRe.FindString(cells[0])
			if id != "" && strings.TrimSpace(cells[1]) != "" && strings.TrimSpace(cells[2]) != "" {
				ev[id] = true
			}
		}
	}
	f, t := section(lines, "Tareas")
	if f < 0 {
		return append(ps, Problem{1, "falta la sección '## Tareas'"})
	}
	for i := f; i < t; i++ {
		m := checkedRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		id := idRe.FindString(m[1])
		switch {
		case id == "":
			ps = append(ps, Problem{i + 1, "tarea marcada [x] sin identificador T-n"})
		case !ev[id]:
			ps = append(ps, Problem{i + 1, fmt.Sprintf("%s está [x] sin fila en Evidencia (| %s | comando | resultado observado | fecha |)", id, id)})
		}
	}
	return ps
}

// CheckAll valida todos los archivos de odd/tasks bajo root.
func CheckAll(root string) (map[string][]Problem, int, error) {
	files, err := filepath.Glob(filepath.Join(root, "odd", "tasks", "*.md"))
	if err != nil {
		return nil, 0, err
	}
	out := map[string][]Problem{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, 0, err
		}
		if ps := Check(string(b)); len(ps) > 0 {
			out[f] = ps
		}
	}
	return out, len(files), nil
}
