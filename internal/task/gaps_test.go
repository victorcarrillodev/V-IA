package task

import (
	"os"
	"strings"
	"testing"
	"time"
)

const sheet = "---\nstatus: %s\n---\n## Specs\n- S1: x\n## Tareas\n%s\n## Evidencia\n| T | c | r |\n|---|---|---|\n| T-1 | go test | ok |\n## Log\n- L1: x\n%s\n"

func problems(status, tasks, log string) string {
	s := strings.Replace(sheet, "%s", status, 1)
	s = strings.Replace(s, "%s", tasks, 1)
	s = strings.Replace(s, "%s", log, 1)
	var out []string
	for _, p := range Check(s) {
		out = append(out, p.Msg)
	}
	return strings.Join(out, "\n")
}

func TestDuplicateTaskID(t *testing.T) {
	got := problems("draft", "- [x] T-1 (S1) a\n- [ ] T-1 (S1) b", "")
	if !strings.Contains(got, "T-1 está repetido") {
		t.Errorf("got %q", got)
	}
}

func TestDoneRequiresAllTasksAndRisk(t *testing.T) {
	got := problems("done", "- [x] T-1 (S1) a\n- [ ] T-2 (S1) b", "")
	for _, want := range []string{"status done con tareas sin cerrar", "falta una línea 'Risk:"} {
		if !strings.Contains(got, want) {
			t.Errorf("falta %q en %q", want, got)
		}
	}
	if got := problems("done", "- [x] T-1 (S1) a", "- Risk: none"); got != "" {
		t.Errorf("ficha cerrada correcta: %q", got)
	}
}

// La petición se cita literal: sin juntar líneas ni espacios.
func TestRequestKeptVerbatim(t *testing.T) {
	req := "falla con:\nError:  code 2\n- [ ] no soy tarea"
	p, err := New(t.TempDir(), "f", "## Specs\n- S1: <pega aquí la petición original, literal>\n", time.Now(), req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if want := "- S1: falla con:\n  > Error:  code 2\n  > - [ ] no soy tarea\n"; !strings.Contains(string(b), want) {
		t.Errorf("got:\n%s", b)
	}
}
