package task

import (
	"os"
	"strings"
	"testing"
	"time"
)

const base = "---\nfeature: x\n---\n## Tareas\n%s\n## Evidencia\n| Tarea | Comando | Resultado | Fecha |\n|---|---|---|---|\n%s\n"

func doc(tasks, ev string) string {
	return strings.Replace(strings.Replace(base, "%s", tasks, 1), "%s", ev, 1)
}

func TestXSinEvidenciaFalla(t *testing.T) {
	ps := Check(doc("- [x] T-1 hacer", ""))
	if len(ps) != 1 || !strings.Contains(ps[0].Msg, "T-1") {
		t.Fatalf("%v", ps)
	}
}

func TestXConEvidenciaPasa(t *testing.T) {
	if ps := Check(doc("- [x] T-1 hacer", "| T-1 | go test | ok | hoy |")); len(ps) != 0 {
		t.Fatalf("%v", ps)
	}
}

func TestEvidenciaVaciaNoCuenta(t *testing.T) {
	if ps := Check(doc("- [x] T-1 hacer", "| T-1 |  |  | hoy |")); len(ps) != 1 {
		t.Fatalf("%v", ps)
	}
}

func TestPendienteNoRequiereEvidencia(t *testing.T) {
	if ps := Check(doc("- [ ] T-1 hacer", "")); len(ps) != 0 {
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
