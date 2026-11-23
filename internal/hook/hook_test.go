package hook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"v-ia/internal/review"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		p    string
		mode string
		del  bool
	}{
		{"Explícame cómo funciona el login", ModeRead, false},
		{"Propón mejoras sin modificar archivos", ModeRead, false},
		{"explica y corrige el bug", ModeEdit, false},
		{"corrige el texto del botón; sin modificar nada más", ModeRead, false},
		{"Corrige el texto del botón X", ModeEdit, false},
		{"haz commit", ModeEdit, true},
		{"arregla esto pero sin hacer commit", ModeEdit, false},
	}
	for _, c := range cases {
		m, d := Classify(c.p)
		if m != c.mode || d != c.del {
			t.Errorf("%q => %s,%v; quería %s,%v", c.p, m, d, c.mode, c.del)
		}
	}
}

func prompt(root, text string) Result {
	return Handle(Input{CWD: root, Event: "UserPromptSubmit", Prompt: text})
}

func edit(root string) Result {
	in := Input{CWD: root, Event: "PreToolUse", Tool: "Edit"}
	in.ToolInput.FilePath = filepath.Join(root, "a.go")
	return Handle(in)
}

func bash(root, cmd string) Result {
	in := Input{CWD: root, Event: "PreToolUse", Tool: "Bash"}
	in.ToolInput.Command = cmd
	return Handle(in)
}

func TestReadModeBlocksEditAndDetectsChanges(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a.go"), []byte("x"), 0o644)
	prompt(root, "explícame cómo funciona esto")
	if !edit(root).Block {
		t.Fatal("el modo lectura debe bloquear Edit")
	}
	os.WriteFile(filepath.Join(root, "a.go"), []byte("y"), 0o644) // edición por otra vía (p. ej. Bash)
	r := Handle(Input{CWD: root, Event: "Stop"})
	if !r.Block {
		t.Fatal("Stop debe bloquear si hubo cambios en modo lectura")
	}
	if Handle(Input{CWD: root, Event: "Stop", StopHookActive: true}).Block {
		t.Fatal("con stop_hook_active no debe volver a bloquear")
	}
}

func TestEditModeAllowsEdit(t *testing.T) {
	root := t.TempDir()
	prompt(root, "corrige el texto del botón")
	if edit(root).Block {
		t.Fatal("modo edición no debe bloquear")
	}
}

func TestDeliveryNeedsExplicitRequest(t *testing.T) {
	root := t.TempDir()
	prompt(root, "corrige el bug")
	if !bash(root, "git commit -m x").Block || !bash(root, "git -C . push").Block || !bash(root, "gh pr create").Block {
		t.Fatal("entrega sin petición explícita debe bloquearse")
	}
	if bash(root, "git status").Block {
		t.Fatal("git status no es entrega")
	}
	prompt(root, "haz commit")
	if bash(root, "git commit -m x").Block {
		t.Fatal("con petición explícita debe permitir commit")
	}
	prompt(root, "sigue con lo otro")
	if !bash(root, "git commit -m x").Block {
		t.Fatal("la autorización no debe persistir al siguiente mensaje")
	}
}

func TestTrackingRequiresTaskFile(t *testing.T) {
	root := t.TempDir()
	SaveState(root, State{Mode: ModeTracking})
	if !edit(root).Block {
		t.Fatal("seguimiento sin ficha debe bloquear")
	}
	in := Input{CWD: root, Event: "PreToolUse", Tool: "Write"}
	in.ToolInput.FilePath = "odd/tasks/f.md"
	if Handle(in).Block {
		t.Fatal("crear la ficha debe estar permitido")
	}
	os.MkdirAll(filepath.Join(root, "odd", "tasks"), 0o755)
	os.WriteFile(filepath.Join(root, "odd", "tasks", "f.md"), []byte("---\n---\n## Tareas\n"), 0o644)
	if edit(root).Block {
		t.Fatal("con ficha debe permitir editar")
	}
	prompt(root, "sigue implementándolo")
	if LoadState(root).Mode != ModeTracking {
		t.Fatal("el seguimiento debe persistir entre turnos")
	}
}

func TestStopBlocksTaskWithoutEvidence(t *testing.T) {
	root := t.TempDir()
	d := filepath.Join(root, "odd", "tasks")
	os.MkdirAll(d, 0o755)
	os.WriteFile(filepath.Join(d, "f.md"), []byte("---\nfeature: f\n---\n## Tareas\n- [x] T-1 hecho\n## Evidencia\n"), 0o644)
	if !Handle(Input{CWD: root, Event: "Stop"}).Block {
		t.Fatal("[x] sin evidencia debe bloquear el cierre")
	}
}

func TestDangerousCommandsBlocked(t *testing.T) {
	root := t.TempDir()
	prompt(root, "haz commit y push")
	for _, c := range []string{"rm -rf /", "rm -rf ~", "rm -fr .", "sudo rm -rf $HOME", "git push --force origin main", "git push -f", "git reset --hard HEAD~1", "git clean -fd", "chmod -R 777 /"} {
		if !bash(root, c).Block {
			t.Errorf("debía bloquear: %s", c)
		}
	}
	for _, c := range []string{"rm -rf build/", "rm file.txt", "git push origin main", "git reset --soft HEAD~1", "git status"} {
		if bash(root, c).Block {
			t.Errorf("no debía bloquear: %s", c)
		}
	}
}

func TestDeliveryNeedsReceiptWhenChecksExist(t *testing.T) {
	root := t.TempDir()
	t.Setenv("V_IA_HOME", t.TempDir())
	os.MkdirAll(filepath.Join(root, ".v-ia"), 0o755)
	os.WriteFile(filepath.Join(root, ".v-ia", "checks.txt"), []byte("true\n"), 0o644)
	prompt(root, "haz commit")
	if !bash(root, "git commit -m x").Block {
		t.Fatal("sin recibo no se debe poder commitear")
	}
	if _, err := review.Run(root, review.DetectCommands(root), time.Minute); err != nil {
		t.Fatal(err)
	}
	if r := bash(root, "git commit -m x"); r.Block {
		t.Fatalf("con recibo válido debe permitir: %s", r.Msg)
	}
	review.SetMode(root, os.Getenv("V_IA_HOME"), "project", false)
	os.WriteFile(filepath.Join(root, "nuevo.txt"), []byte("x"), 0o644)
	if bash(root, "git commit -m x").Block {
		t.Fatal("con RDD apagado no se exige recibo")
	}
}

func TestSessionStartRecoversEngramContextAndRefreshesRegistry(t *testing.T) {
	bin, root := t.TempDir(), t.TempDir()
	t.Setenv("V_IA_HOME", t.TempDir())
	fake := "#!/bin/sh\nif [ \"$1\" = context ]; then echo 'decisión previa: usar sqlite'; fi\n"
	if err := os.WriteFile(filepath.Join(bin, "engram"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	r := Handle(Input{CWD: root, Event: "SessionStart"})
	if r.Block || !strings.Contains(r.Msg, "decisión previa: usar sqlite") || !strings.Contains(r.Msg, "gana el código") {
		t.Fatalf("debe inyectar la memoria con la regla de contraste: %q", r.Msg)
	}
	os.MkdirAll(filepath.Join(root, ".atl"), 0o755)
	os.WriteFile(filepath.Join(root, ".atl", "skill-registry.md"), []byte("viejo"), 0o644)
	if r := Handle(Input{CWD: root, Event: "SessionStart"}); !strings.Contains(r.Msg, "skill-registry.md") {
		t.Fatalf("debe apuntar al registro: %q", r.Msg)
	}
	if b, _ := os.ReadFile(filepath.Join(root, ".atl", "skill-registry.md")); string(b) == "viejo" {
		t.Fatal("el registro debía refrescarse")
	}
}

func TestSessionStartWithoutEngramIsSilent(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("V_IA_HOME", t.TempDir())
	if r := Handle(Input{CWD: t.TempDir(), Event: "SessionStart"}); r.Block || r.Msg != "" {
		t.Fatalf("sin engram ni registro no debe decir nada: %+v", r)
	}
}
