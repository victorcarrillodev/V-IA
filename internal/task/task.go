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
	specRe    = regexp.MustCompile(`\bS\d+\b`)
	logRe     = regexp.MustCompile(`^\s*[-*]\s*L1\s*:\s*(.*)$`)
	anyTaskRe = regexp.MustCompile(`^\s*- \[[ xX]\]\s*(.*)$`)
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
// request, si no está vacío, se escribe literal en S1 y en L1.
func New(root, feature, tmpl string, now time.Time, request ...string) (string, error) {
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
	if len(request) > 0 && strings.TrimSpace(request[0]) != "" {
		line := strings.Join(strings.Fields(request[0]), " ")
		tmpl = strings.NewReplacer(
			"<pega aquí la petición literal del usuario: textos, mensajes de error y ejemplos exactos>", line,
			"<pega aquí la petición original, literal>", line).Replace(tmpl)
	}
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
	specs := map[string]bool{}
	if f, t := section(lines, "Specs"); f < 0 {
		ps = append(ps, Problem{1, "falta la sección '## Specs' con S1: la petición literal del usuario"})
	} else {
		for _, l := range lines[f:t] {
			if strings.Contains(l, "<pega") {
				ps = append(ps, Problem{f, "Specs sin completar: pega la petición literal del usuario"})
				break
			}
		}
		for _, l := range lines[f:t] {
			if id := specRe.FindString(l); id != "" && strings.HasPrefix(strings.TrimSpace(l), "-") {
				specs[id] = true
			}
		}
		if len(specs) == 0 {
			ps = append(ps, Problem{f, "Specs sin ningún S# (ej. '- S1: ...')"})
		}
	}
	if f, t := section(lines, "Log"); f < 0 {
		ps = append(ps, Problem{1, "falta la sección '## Log' con L1: la petición original"})
	} else {
		ok := false
		for _, l := range lines[f:t] {
			if m := logRe.FindStringSubmatch(l); m != nil && strings.TrimSpace(m[1]) != "" && !strings.Contains(m[1], "<pega") {
				ok = true
			}
		}
		if !ok {
			ps = append(ps, Problem{f, "Log sin L1: pega la petición original literal"})
		}
	}
	f, t := section(lines, "Tareas")
	if f < 0 {
		return append(ps, Problem{1, "falta la sección '## Tareas'"})
	}
	for i := f; i < t; i++ {
		if a := anyTaskRe.FindStringSubmatch(lines[i]); a != nil && len(specs) > 0 {
			refs := specRe.FindAllString(a[1], -1)
			if len(refs) == 0 {
				ps = append(ps, Problem{i + 1, "tarea sin spec enlazada (S#)"})
			}
			for _, r := range refs {
				if !specs[r] {
					ps = append(ps, Problem{i + 1, "la tarea cita " + r + ", que no existe en Specs"})
				}
			}
		}
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

// Summary cuenta tareas hechas y totales de un documento.
func Summary(content string) (done, total int) {
	lines := strings.Split(content, "\n")
	f, t := section(lines, "Tareas")
	if f < 0 {
		return 0, 0
	}
	for _, l := range lines[f:t] {
		if m := anyTaskRe.FindStringSubmatch(l); m != nil {
			total++
			if checkedRe.MatchString(l) {
				done++
			}
		}
	}
	return
}

// Status describe cada ficha: feature, estado y avance.
func Status(root string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(root, "odd", "tasks", "*.md"))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		st := "?"
		for _, l := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(l, "status:"); ok {
				st = strings.TrimSpace(v)
				break
			}
		}
		d, n := Summary(string(b))
		out = append(out, fmt.Sprintf("%s  [%s]  %d/%d tareas", filepath.Base(f), st, d, n))
	}
	return out, nil
}

// EvidenceCommands devuelve, por tarea [x], el comando que declara su fila de Evidencia.
func EvidenceCommands(content string) map[string]string {
	lines := strings.Split(content, "\n")
	out := map[string]string{}
	f, t := section(lines, "Evidencia")
	if f < 0 {
		return out
	}
	for _, l := range lines[f:t] {
		if !strings.HasPrefix(strings.TrimSpace(l), "|") {
			continue
		}
		cells := strings.Split(strings.Trim(strings.TrimSpace(l), "|"), "|")
		if len(cells) < 3 {
			continue
		}
		if id := idRe.FindString(cells[0]); id != "" {
			out[id] = strings.TrimSpace(strings.ReplaceAll(cells[1], "`", ""))
		}
	}
	return out
}

// Verify contrasta la evidencia de cada tarea [x] con los comandos que el recibo vigente
// ejecutó de verdad y pasaron (ran). Una tarea cuya evidencia no cita ninguno de ellos
// queda "sin respaldo": no es un error de forma, pero tampoco está demostrada por V-IA.
// Devuelve las tareas respaldadas y las que no.
func Verify(content string, ran []string) (backed, unbacked []string) {
	lines := strings.Split(content, "\n")
	f, t := section(lines, "Tareas")
	if f < 0 {
		return nil, nil
	}
	ev := EvidenceCommands(content)
	for _, l := range lines[f:t] {
		m := checkedRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		id := idRe.FindString(m[1])
		if id == "" {
			continue
		}
		ok := false
		for _, c := range ran {
			if n := normCmd(c); n != "" && strings.Contains(normCmd(ev[id]), n) {
				ok = true
				break
			}
		}
		if ok {
			backed = append(backed, id)
		} else {
			unbacked = append(unbacked, id)
		}
	}
	return
}

// normCmd deja un comando sin banderas (-count=1, -race, -q…) y con espacios simples, para
// que "go test -count=1 ./..." respalde a "go test ./...".
func normCmd(s string) string {
	var out []string
	for _, f := range strings.Fields(s) {
		if !strings.HasPrefix(f, "-") {
			out = append(out, f)
		}
	}
	return strings.Join(out, " ")
}
