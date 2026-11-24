package hook

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWritesFiles(t *testing.T) {
	yes := []string{
		"sed -i 's/a/b/' main.go", "sed -n 1p x && sed -i.bak s/a/b/ f", "echo hola > hello.go", "cat x >> notas.md",
		"printf x | tee out.txt", "mv a.go b.go", "rm hello.go", "touch nuevo", "git apply fix.patch",
		"git checkout -- main.go", "git restore main.go", "perl -pi -e 's/a/b/' f", "ls && mkdir build", "cp a b",
		"bash -lc 'echo x > f.txt'", "sh -ec \"cat > a\"", "eval \"echo x > f.txt\"", "ssh h \"cat > f\"", "su -c \"rm -r x\"", "sudo -u bob rm x", "find . -name '*.go' -delete", "find . -type f -exec rm {} +", "ls | xargs rm", "env X=1 rm a.go", "python3 -c \"open('a.go','w').write('x')\"", "bash -c 'echo x > a.go'", "git stash", "cp /tmp/x.go internal/foo.go", "mv /tmp/a main.go", "touch /tmp/a src/b", "rm /tmp/x importante.txt",
	}
	no := []string{
		"ls -la", "cat main.go", "grep -rn foo . 2>&1", "go test ./... 2>&1 | tail -n 5", "git status --porcelain",
		"git diff --stat", "v-ia guard start", "echo hi > /tmp/x.txt", "make 2>/dev/null", "ls > /dev/null", "git log --oneline | head",
		"bash -lc 'ls -la'", "eval \"echo hi\"", "ssh h \"uptime\"", "awk '$1>3' f", "grep \"a>b\" f", "git stash list", "find . -name '*.go'", "xargs echo", "python3 -c \"print(1)\"", "bash -c 'ls'", "go build ./... &>/dev/null", "rm -f /tmp/algo.txt", "sed -n '1,5p' main.go", "mkdir -p /tmp/x/y",
	}
	for _, c := range yes {
		if !writesFiles(c) {
			t.Errorf("debía detectar escritura: %q", c)
		}
	}
	for _, c := range no {
		if writesFiles(c) {
			t.Errorf("falso positivo: %q", c)
		}
	}
}

func TestReadModeBlocksBashWrites(t *testing.T) {
	root := t.TempDir()
	_ = SaveState(root, State{Mode: ModeRead})
	in := Input{CWD: root, Event: "PreToolUse", Tool: "Bash"}
	in.ToolInput.Command = "sed -i 's/a/b/' main.go"
	if r := Handle(in); !r.Block {
		t.Fatal("sed -i en modo lectura debe bloquearse al instante")
	}
	in.ToolInput.Command = "grep -rn x ."
	if r := Handle(in); r.Block {
		t.Fatal("grep en modo lectura no debe bloquearse")
	}
	_ = SaveState(root, State{Mode: ModeEdit})
	in.ToolInput.Command = "sed -i 's/a/b/' main.go"
	if r := Handle(in); r.Block {
		t.Fatal("sed -i en modo edición está permitido")
	}
}

func TestTDDStrict(t *testing.T) {
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-C", root}, args...)...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run("init", "-q")
	_ = SaveState(root, State{Mode: ModeEdit})
	edit := func(p string) Result {
		in := Input{CWD: root, Event: "PreToolUse", Tool: "Edit"}
		in.ToolInput.FilePath = filepath.Join(root, p)
		return Handle(in)
	}
	if edit("a.go").Block {
		t.Fatal("sin TDD estricto no se bloquea")
	}
	if err := SetTDD(root, true); err != nil {
		t.Fatal(err)
	}
	if !edit("a.go").Block {
		t.Fatal("TDD estricto: código sin prueba debe bloquearse")
	}
	if edit("a_test.go").Block || edit("README.md").Block || edit("go.mod").Block {
		t.Fatal("pruebas, docs y configuración no se bloquean")
	}
	if err := os.WriteFile(filepath.Join(root, "a_test.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if edit("a.go").Block {
		t.Fatal("con una prueba nueva ya se puede editar el código")
	}
}

func TestIsTestPath(t *testing.T) {
	for _, p := range []string{"a_test.go", "x/y.test.ts", "src/a.spec.js", "tests/foo.py", "pkg/test_x.py", "src/__tests__/a.ts"} {
		if !IsTestPath(p) {
			t.Errorf("%s es prueba", p)
		}
	}
	for _, p := range []string{"main.go", "src/latest.go", "contest.py", "internal/hook/hook.go"} {
		if IsTestPath(p) {
			t.Errorf("%s no es prueba", p)
		}
	}
}

func BenchmarkHandleBash(b *testing.B) {
	root := b.TempDir()
	_ = SaveState(root, State{Mode: ModeRead})
	in := Input{CWD: root, Event: "PreToolUse", Tool: "Bash"}
	in.ToolInput.Command = "go test ./... 2>&1 | tail -n 5"
	for i := 0; i < b.N; i++ {
		Handle(in)
	}
}

func TestGeminiEventsAndToolsAreNormalized(t *testing.T) {
	root := t.TempDir()
	_ = SaveState(root, State{Mode: ModeRead})
	in := Input{CWD: root, Event: "BeforeTool", Tool: "write_file"}
	in.ToolInput.FilePath = "a.go"
	if r := Handle(in); !r.Block {
		t.Fatal("write_file de Gemini en modo lectura debe bloquearse")
	}
	in = Input{CWD: root, Event: "BeforeTool", Tool: "run_shell_command"}
	in.ToolInput.Command = "echo x > a.go"
	if r := Handle(in); !r.Block {
		t.Fatal("run_shell_command que escribe debe bloquearse en modo lectura")
	}
	in = Input{CWD: root, Event: "BeforeAgent", Prompt: "Corrige el botón"}
	Handle(in)
	if LoadState(root).Mode != ModeEdit {
		t.Fatal("BeforeAgent debe clasificar la petición")
	}
}

func TestCodexAndCursorShapesAreNormalized(t *testing.T) {
	root := t.TempDir()
	_ = SaveState(root, State{Mode: ModeRead})
	// Codex: apply_patch y Bash
	in := Input{CWD: root, Event: "PreToolUse", Tool: "apply_patch"}
	if r := Handle(in); !r.Block {
		t.Fatal("apply_patch de Codex en modo lectura debe bloquearse")
	}
	in = Input{CWD: root, Event: "PreToolUse", Tool: "Bash"}
	in.ToolInput.Command = "rm -rf /"
	if r := Handle(in); !r.Block {
		t.Fatal("rm -rf / debe bloquearse en Codex")
	}
	// Cursor: preToolUse con Shell / Write / Delete
	for _, tool := range []string{"Write", "Delete"} {
		in = Input{CWD: root, Event: "preToolUse", Tool: tool}
		in.ToolInput.FilePath = "a.go"
		if r := Handle(in); !r.Block {
			t.Fatalf("%s de Cursor en modo lectura debe bloquearse", tool)
		}
	}
	in = Input{CWD: root, Event: "preToolUse", Tool: "Shell"}
	in.ToolInput.Command = "echo x > a.go"
	if r := Handle(in); !r.Block {
		t.Fatal("Shell de Cursor que escribe debe bloquearse en modo lectura")
	}
	in.ToolInput.Command = "ls"
	if r := Handle(in); r.Block {
		t.Fatal("ls no debe bloquearse")
	}
}

func TestProtectedStateCannotBeEditedByAgent(t *testing.T) {
	root := t.TempDir()
	_ = SaveState(root, State{Mode: ModeEdit})
	edit := Input{CWD: root, Event: "PreToolUse", Tool: "Edit"}
	for _, f := range []string{".v-ia/receipt.json", filepath.Join(root, ".v-ia", "review-mode"), ".v-ia/state.json", ".v-ia/tdd"} {
		edit.ToolInput.FilePath = f
		if r := Handle(edit); !r.Block {
			t.Errorf("%s no debe ser editable por el agente", f)
		}
	}
	edit.ToolInput.FilePath = ".v-ia/checks.txt"
	if r := Handle(edit); r.Block {
		t.Error("checks.txt es configuración del usuario: editable")
	}
	bash := Input{CWD: root, Event: "PreToolUse", Tool: "Bash"}
	bash.ToolInput.Command = "echo '{}' > .v-ia/receipt.json"
	if r := Handle(bash); !r.Block {
		t.Error("escribir el recibo por Bash debe bloquearse incluso en modo edición")
	}
	bash.ToolInput.Command = "v-ia review reset"
	if r := Handle(bash); r.Block {
		t.Error("v-ia sí puede gestionar su estado")
	}
	bash.ToolInput.Command = "cat .v-ia/receipt.json"
	if r := Handle(bash); r.Block {
		t.Error("leer el recibo está permitido")
	}
}

func TestStopRemindsOncePerCandidateOnHighRisk(t *testing.T) {
	root := t.TempDir()
	t.Setenv("V_IA_HOME", t.TempDir())
	for _, a := range [][]string{{"init", "-q"}} {
		if err := exec.Command("git", append([]string{"-C", root}, a...)...).Run(); err != nil {
			t.Fatal(err)
		}
	}
	_ = os.MkdirAll(filepath.Join(root, ".v-ia"), 0o755)
	_ = os.WriteFile(filepath.Join(root, ".v-ia", "checks.txt"), []byte("true\n"), 0o644)
	_ = os.MkdirAll(filepath.Join(root, "internal", "auth"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "internal", "auth", "login.go"), []byte("package auth\n"), 0o644)
	_ = SaveState(root, State{Mode: ModeEdit})
	r := Handle(Input{CWD: root, Event: "Stop"})
	if !r.Block || !strings.Contains(r.Msg, "revisión independiente") {
		t.Fatalf("riesgo alto sin revisión debe recordarse al cerrar: %+v", r)
	}
	if r := Handle(Input{CWD: root, Event: "Stop"}); r.Block {
		t.Fatal("el aviso es único por candidato")
	}
}
