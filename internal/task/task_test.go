package task

import (
	"os"
	"strings"
	"testing"
	"time"
)

const base = "---\nfeature: x\n---\n## Specs\n- S1: pide X literal\n\n## Tareas\n%s\n## Evidencia\n| Tarea | Comando | Resultado | Fecha |\n|---|---|---|---|\n%s\n## Log\n- L1: pide X literal\n"

func doc(tasks, ev string) string {
	return strings.Replace(strings.Replace(base, "%s", tasks, 1), "%s", ev, 1)
}

func TestXSinEvidenciaFalla(t *testing.T) {
	ps := Check(doc("- [x] T-1 (S1) hacer", ""))
	if len(ps) != 1 || !strings.Contains(ps[0].Msg, "T-1") {
		t.Fatalf("%v", ps)
	}
}

func TestXConEvidenciaPasa(t *testing.T) {
	if ps := Check(doc("- [x] T-1 (S1) hacer", "| T-1 | go test | ok | hoy |")); len(ps) != 0 {
		t.Fatalf("%v", ps)
	}
}

func TestEvidenciaVaciaNoCuenta(t *testing.T) {
	if ps := Check(doc("- [x] T-1 (S1) hacer", "| T-1 |  |  | hoy |")); len(ps) != 1 {
		t.Fatalf("%v", ps)
	}
}

func TestPendienteNoRequiereEvidencia(t *testing.T) {
	if ps := Check(doc("- [ ] T-1 (S1) hacer", "")); len(ps) != 0 {
		t.Fatalf("%v", ps)
	}
}

func TestNewNoSobrescribe(t *testing.T) {
	d := t.TempDir()
	p, err := New(d, "Mi Feature Ñ", "feature: <nombre>\n<fecha>", time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC))
	if err != nil || !strings.HasSuffix(p, "mi-feature-ñ.md") {
		t.Fatalf("%q %v", p, err)
	}
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), "Mi Feature Ñ") || !strings.Contains(string(b), "2026-11-23") {
		t.Fatalf("%s", b)
	}
	if _, err := New(d, "Mi Feature Ñ", "", time.Now()); err == nil {
		t.Fatal("debía negarse a sobrescribir")
	}
}

func TestSpecsLogYEnlaces(t *testing.T) {
	sin := "---\nfeature: x\n---\n## Tareas\n- [ ] T-1 hacer\n"
	if ps := Check(sin); len(ps) < 2 {
		t.Fatalf("debe exigir Specs y Log: %v", ps)
	}
	if ps := Check(doc("- [ ] T-1 hacer", "")); len(ps) != 1 || !strings.Contains(ps[0].Msg, "spec") {
		t.Fatalf("tarea sin S#: %v", ps)
	}
	if ps := Check(doc("- [ ] T-1 (S9) hacer", "")); len(ps) != 1 || !strings.Contains(ps[0].Msg, "S9") {
		t.Fatalf("S# inexistente: %v", ps)
	}
}

func TestPlantillaSinCompletarFallaYConPeticionPasa(t *testing.T) {
	tmpl, _ := os.ReadFile("../../core/odd/templates/task.md")
	d := t.TempDir()
	p, _ := New(d, "feat uno", string(tmpl), time.Now())
	b, _ := os.ReadFile(p)
	if len(Check(string(b))) == 0 {
		t.Fatal("la plantilla sin completar no debe validar")
	}
	p2, _ := New(d, "feat dos", string(tmpl), time.Now(), "Crea el comando  foo\ny que imprima \"hola\"")
	b2, _ := os.ReadFile(p2)
	if ps := Check(string(b2)); len(ps) != 0 {
		t.Fatalf("con petición literal debe validar: %v\n%s", ps, b2)
	}
	if !strings.Contains(string(b2), `S1: Crea el comando foo y que imprima "hola"`) {
		t.Fatalf("S1 no es literal:\n%s", b2)
	}
}

func TestStatus(t *testing.T) {
	d := t.TempDir()
	os.MkdirAll(d+"/odd/tasks", 0o755)
	os.WriteFile(d+"/odd/tasks/a.md", []byte(doc("- [x] T-1 (S1) a\n- [ ] T-2 (S1) b", "| T-1 | c | r | h |")), 0o644)
	out, _ := Status(d)
	if len(out) != 1 || !strings.Contains(out[0], "1/2") {
		t.Fatalf("%v", out)
	}
}

func TestVerifyEvidenceAgainstReceipt(t *testing.T) {
	doc := "---\nfeature: x\n---\n## Specs\n- S1: algo\n\n## Tareas\n- [x] T-1 (S1) hecho\n- [x] T-2 (S1) hecho\n- [ ] T-3 (S1) pendiente\n\n## Evidencia\n| Tarea | Comando / comprobación | Resultado | Fecha |\n|---|---|---|---|\n| T-1 | `go test ./...` | ok | hoy |\n| T-2 | lo miré a ojo | bien | hoy |\n\n## Log\n- L1: algo\n"
	backed, unbacked := Verify(doc, []string{"go build ./...", "go test ./..."})
	if len(backed) != 1 || backed[0] != "T-1" || len(unbacked) != 1 || unbacked[0] != "T-2" {
		t.Fatalf("backed=%v unbacked=%v", backed, unbacked)
	}
	if b, u := Verify(doc, nil); len(b) != 0 || len(u) != 2 {
		t.Fatalf("sin comandos ejecutados nada está respaldado: %v %v", b, u)
	}
}

func TestVerifyIgnoresFlags(t *testing.T) {
	doc := "---\nfeature: x\n---\n## Specs\n- S1: a\n\n## Tareas\n- [x] T-1 (S1) hecho\n\n## Evidencia\n| Tarea | Comando | Resultado | Fecha |\n|---|---|---|---|\n| T-1 | go test -count=1 -race ./... (todo) | ok | hoy |\n\n## Log\n- L1: a\n"
	if b, _ := Verify(doc, []string{"go test ./..."}); len(b) != 1 {
		t.Fatal("las banderas no deben impedir el respaldo")
	}
	if b, _ := Verify(doc, []string{"go vet ./..."}); len(b) != 0 {
		t.Fatal("otro comando no respalda")
	}
}
