package hook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	corefs "v-ia/core"
	"v-ia/internal/review"
)

// Escenarios E1–E7 del plan, ejecutados contra las reglas reales de los hooks.
// Verifican el mecanismo (determinista); el comportamiento del modelo se prueba con scripts/e2e-claude.sh.

func newProj(t *testing.T) string {
	t.Setenv("V_IA_HOME", t.TempDir())
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "login.go"), []byte("package x\n"), 0o644)
	return root
}

func TestE1_E2_LecturaNoModificaNada(t *testing.T) {
	for _, p := range []string{"Explícame cómo funciona el login", "Propón mejoras sin modificar archivos"} {
		root := newProj(t)
		prompt(root, p)
		if !edit(root).Block {
			t.Errorf("%q: la edición debía bloquearse", p)
		}
		if Handle(Input{CWD: root, Event: "Stop"}).Block {
			t.Errorf("%q: sin cambios, el cierre debe pasar", p)
		}
	}
}

func TestE3_AcotadoPermiteEditar(t *testing.T) {
	root := newProj(t)
	prompt(root, "Corrige el texto del botón X")
	if edit(root).Block {
		t.Fatal("un cambio acotado debe poder editar")
	}
}

func TestE4_SeguimientoExigeFichaAntesDeEditar(t *testing.T) {
	root := newProj(t)
	SaveState(root, State{Mode: ModeTracking})
	if !edit(root).Block {
		t.Fatal("sin ficha no se edita")
	}
	tmpl, _ := corefs.FS.ReadFile("odd/templates/task.md")
	os.MkdirAll(filepath.Join(root, "odd", "tasks"), 0o755)
	os.WriteFile(filepath.Join(root, "odd", "tasks", "f.md"), tmpl, 0o644)
	if edit(root).Block {
		t.Fatal("con ficha se puede editar")
	}
	if !Handle(Input{CWD: root, Event: "Stop"}).Block {
		t.Fatal("la ficha sin completar (petición literal) no debe permitir cerrar")
	}
}

func TestE5_RetomarContrastaConElCodigo(t *testing.T) {
	b, err := corefs.FS.ReadFile("skills/odd-resume/SKILL.md")
	if err != nil || !strings.Contains(string(b), "gana el código") || !strings.Contains(string(b), "odd/tasks/") {
		t.Fatalf("odd-resume debe leer la ficha y contrastar con el código: %v", err)
	}
}

func TestE6_MarcarHechoSinEvidenciaSeRechaza(t *testing.T) {
	root := newProj(t)
	d := filepath.Join(root, "odd", "tasks")
	os.MkdirAll(d, 0o755)
	doc := "---\nfeature: f\n---\n## Specs\n- S1: pide X\n\n## Tareas\n- [x] T-1 (S1) hecho\n\n## Evidencia\n\n## Log\n- L1: pide X\n"
	os.WriteFile(filepath.Join(d, "f.md"), []byte(doc), 0o644)
	r := Handle(Input{CWD: root, Event: "Stop"})
	if !r.Block || !strings.Contains(r.Msg, "T-1") {
		t.Fatalf("debe bloquear por falta de evidencia: %+v", r)
	}
}

func TestE7_CommitTrasReciboSoloSiElUsuarioLoPidio(t *testing.T) {
	root := newProj(t)
	os.MkdirAll(filepath.Join(root, ".v-ia"), 0o755)
	os.WriteFile(filepath.Join(root, ".v-ia", "checks.txt"), []byte("true\n"), 0o644)
	if _, err := review.Run(root, review.DetectCommands(root), time.Minute); err != nil {
		t.Fatal(err)
	}
	prompt(root, "sigue con lo demás") // recibo pass, pero nadie pidió commit
	if !bash(root, "git commit -m x").Block {
		t.Fatal("un recibo pass no autoriza commit")
	}
	prompt(root, "haz commit")
	if bash(root, "git commit -m x").Block {
		t.Fatal("con petición explícita y recibo válido debe permitir")
	}
}
