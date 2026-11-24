package standards

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTemplateEsValida(t *testing.T) {
	rs, err := Parse(Template)
	if err != nil || len(rs) != 3 {
		t.Fatalf("rs=%d err=%v", len(rs), err)
	}
	if rs[0].Kind != Path || rs[1].Kind != Deny || rs[2].Kind != AI {
		t.Fatalf("tipos: %v", rs)
	}
}

func TestParseErrores(t *testing.T) {
	casos := map[string]string{
		"sin type":       "id: a\npaths: **",
		"type raro":      "id: a\ntype: x\npaths: **",
		"deny sin regex": "id: a\ntype: deny\npaths: **",
		"regex mala":     "id: a\ntype: deny\npaths: **\npattern: (",
		"path sin where": "id: a\ntype: path\npaths: **",
		"ai sin text":    "id: a\ntype: ai\npaths: **",
		"sin paths":      "id: a\ntype: ai\ntext: x",
		"id malo":        "id: A B\ntype: ai\npaths: **\ntext: x",
		"línea suelta":   "id: a\nnada",
		"repetida":       "id: a\ntype: ai\npaths: **\ntext: x\n---\nid: a\ntype: ai\npaths: **\ntext: y",
	}
	for n, src := range casos {
		if _, err := Parse(src); err == nil {
			t.Errorf("%s: debía fallar", n)
		}
	}
}

func TestGlobs(t *testing.T) {
	casos := []struct {
		g, p string
		ok   bool
	}{
		{"**/*_test.go", "a_test.go", true},
		{"**/*_test.go", "internal/x/a_test.go", true},
		{"internal/**", "internal/a/b.go", true},
		{"internal/**/*.go", "internal/a.go", true},
		{"internal/*.go", "internal/a/b.go", false},
		{"cmd/**", "internal/a.go", false},
		{"*.md", "docs/a.md", false},
	}
	for _, c := range casos {
		if got := Match([]string{c.g}, c.p); got != c.ok {
			t.Errorf("%q ~ %q = %v, quería %v", c.g, c.p, got, c.ok)
		}
	}
}

const sampleDiff = `diff --git a/internal/a.go b/internal/a.go
index 1..2 100644
--- a/internal/a.go
+++ b/internal/a.go
@@ -1,0 +2,2 @@
+	println("x")
+	ok()
diff --git a/pkg/nuevo_test.go b/pkg/nuevo_test.go
new file mode 100644
--- /dev/null
+++ b/pkg/nuevo_test.go
@@ -0,0 +1 @@
+package pkg
diff --git a/go.sum b/go.sum
--- a/go.sum
+++ b/go.sum
@@ -1 +1,2 @@
+hash
`

func TestParseDiffYDeterministas(t *testing.T) {
	fs := ParseDiff(sampleDiff)
	if len(fs) != 2 { // go.sum se ignora
		t.Fatalf("archivos=%d", len(fs))
	}
	if fs[0].Added[0].N != 2 || fs[0].Added[1].N != 3 || !fs[1].New {
		t.Fatalf("diff mal interpretado: %+v", fs)
	}
	rs, _ := Parse(Template)
	got := Deterministic(rs, fs)
	if len(got) != 2 {
		t.Fatalf("hallazgos=%v", got)
	}
	if got[0].Rule != "sin-println" || got[0].Line != 2 {
		t.Errorf("deny: %v", got[0])
	}
	if got[1].Rule != "tests-junto-al-codigo" || got[1].File != "pkg/nuevo_test.go" {
		t.Errorf("path: %v", got[1])
	}
}

func aiRules() []Rule {
	rs, _ := Parse("id: ctx\ntype: ai\npaths: **/*.go\ntext: envuelve los errores")
	return rs
}

func aiFiles() []File {
	return []File{{Path: "a.go", Added: []Added{{1, "return err"}}}, {Path: "b.go", Added: []Added{{1, "return nil"}}}}
}

func TestReviewAIUnaLlamadaYCache(t *testing.T) {
	root := t.TempDir()
	llamadas := 0
	run := func(p string, _ time.Duration) (string, error) {
		llamadas++
		if !strings.Contains(p, "regla: ctx") || !strings.Contains(p, "return err") {
			t.Errorf("prompt incompleto: %s", p)
		}
		return "```json\n[{\"rule\":\"ctx\",\"file\":\"a.go\",\"line\":1,\"msg\":\"error desnudo\"}]\n```", nil
	}
	fs, st, err := ReviewAI(root, aiRules(), aiFiles(), run, time.Second)
	if err != nil || llamadas != 1 || len(fs) != 1 || fs[0].File != "a.go" || st.Sent != 2 {
		t.Fatalf("fs=%v st=%+v llamadas=%d err=%v", fs, st, llamadas, err)
	}
	// segunda vez: todo sale de la caché, sin llamar al agente (también lo limpio)
	fs, st, err = ReviewAI(root, aiRules(), aiFiles(), run, time.Second)
	if err != nil || llamadas != 1 || len(fs) != 1 || st.Cached != 2 || st.Called {
		t.Fatalf("la caché no se usó: fs=%v st=%+v llamadas=%d", fs, st, llamadas)
	}
	// si cambia la regla, se vuelve a revisar
	rs, _ := Parse("id: ctx\ntype: ai\npaths: **/*.go\ntext: otra cosa distinta")
	if _, st, _ = ReviewAI(root, rs, aiFiles(), run, time.Second); !st.Called || llamadas != 2 {
		t.Fatalf("cambiar la regla debía invalidar la caché: %+v", st)
	}
}

func TestReviewAIFallosNoCacheanNiMienten(t *testing.T) {
	root := t.TempDir()
	for _, run := range []Runner{
		func(string, time.Duration) (string, error) { return "", errors.New("sin agente") },
		func(string, time.Duration) (string, error) { return "no sé", nil },
		func(string, time.Duration) (string, error) { return "[{roto", nil },
	} {
		fs, _, err := ReviewAI(root, aiRules(), aiFiles(), run, time.Second)
		if err == nil || len(fs) != 0 {
			t.Fatalf("debía devolver error y ningún hallazgo: fs=%v err=%v", fs, err)
		}
	}
	if CacheSize(root) != 0 {
		t.Fatal("un fallo no debe dejar nada en caché")
	}
}

func TestReviewAIIgnoraHallazgosAjenos(t *testing.T) {
	run := func(string, time.Duration) (string, error) {
		return `[{"rule":"inventada","file":"a.go","line":1,"msg":"x"},{"rule":"ctx","file":"otro.go","line":1,"msg":"y"}]`, nil
	}
	fs, _, err := ReviewAI(t.TempDir(), aiRules(), aiFiles(), run, time.Second)
	if err != nil || len(fs) != 0 {
		t.Fatalf("no debe aceptar reglas/archivos que no pidió: %v %v", fs, err)
	}
}

func TestReviewAIPresupuesto(t *testing.T) {
	var big []File
	for i := 0; i < 8; i++ {
		big = append(big, File{Path: string(rune('a'+i)) + ".go", Added: []Added{{1, strings.Repeat("x", 5000)}}})
	}
	var prompt string
	run := func(p string, _ time.Duration) (string, error) { prompt = p; return "[]", nil }
	_, st, err := ReviewAI(t.TempDir(), aiRules(), big, run, time.Second)
	if err != nil || st.Skipped == 0 || len(prompt) > maxTotalBytes+4000 {
		t.Fatalf("st=%+v len=%d err=%v", st, len(prompt), err)
	}
}

func gitRepo(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	for _, a := range [][]string{{"init", "-q"}, {"config", "user.email", "t@t"}, {"config", "user.name", "t"}} {
		if out, err := exec.Command("git", append([]string{"-C", d}, a...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", a, out)
		}
	}
	return d
}

func TestHookConservaElExistenteYSeQuita(t *testing.T) {
	d := gitRepo(t)
	hp := filepath.Join(d, ".git", "hooks", "pre-commit")
	os.MkdirAll(filepath.Dir(hp), 0o755)
	os.WriteFile(hp, []byte("#!/bin/sh\necho mio\n"), 0o755)
	if _, err := InstallHook(d, "/bin/v-ia", false); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallHook(d, "/bin/v-ia", true); err != nil { // idempotente y actualiza
		t.Fatal(err)
	}
	b, _ := os.ReadFile(hp)
	if strings.Count(string(b), hookBegin) != 1 || !strings.Contains(string(b), "echo mio") || !strings.Contains(string(b), "--hook --strict") {
		t.Fatalf("hook mal: %s", b)
	}
	if on, strict := HookState(d); !on || !strict {
		t.Fatal("estado esperado: instalado y estricto")
	}
	if ok, err := RemoveHook(d); !ok || err != nil {
		t.Fatal("no se quitó")
	}
	b, _ = os.ReadFile(hp)
	if strings.Contains(string(b), "v-ia") || !strings.Contains(string(b), "echo mio") {
		t.Fatalf("debe conservar lo tuyo y quitar lo nuestro: %s", b)
	}
}

func TestHookPropioSeBorraSiQuedaVacio(t *testing.T) {
	d := gitRepo(t)
	p, _ := InstallHook(d, "/bin/v-ia", false)
	if st, _ := os.Stat(p); st.Mode()&0o111 == 0 {
		t.Fatal("el hook debe ser ejecutable")
	}
	RemoveHook(d)
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("un hook solo nuestro debe borrarse al quitarlo")
	}
}

func TestChangesStaged(t *testing.T) {
	d := gitRepo(t)
	os.WriteFile(filepath.Join(d, "a.go"), []byte("package a\n\nfunc f() { println(1) }\n"), 0o644)
	exec.Command("git", "-C", d, "add", "a.go").Run()
	fs, err := Changes(d, Staged)
	if err != nil || len(fs) != 1 || !fs[0].New || len(fs[0].Added) != 3 {
		t.Fatalf("fs=%+v err=%v", fs, err)
	}
}
