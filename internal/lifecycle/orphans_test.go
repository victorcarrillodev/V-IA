package lifecycle

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	corefs "v-ia/core"
)

// Estas pruebas impiden que algo del proyecto quede huérfano o sin documentar.

func repoRoot(t *testing.T) string {
	t.Helper()
	d, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
		d = filepath.Dir(d)
	}
	t.Fatal("no encuentro la raíz del repositorio")
	return ""
}

func goSources(t *testing.T, root string) string {
	t.Helper()
	var sb strings.Builder
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go") {
			b, _ := os.ReadFile(p)
			sb.Write(b)
		}
		return nil
	})
	return sb.String()
}

// Cada skill embebida se instala, y cada skill instalada existe y se llama como su carpeta.
func TestSkillsEmbebidasCoincidenConLasInstaladas(t *testing.T) {
	entries, err := fs.ReadDir(corefs.FS, "skills")
	if err != nil {
		t.Fatal(err)
	}
	var embedded []string
	for _, e := range entries {
		embedded = append(embedded, e.Name())
		b, err := corefs.FS.ReadFile("skills/" + e.Name() + "/SKILL.md")
		if err != nil {
			t.Errorf("skill %s sin SKILL.md", e.Name())
			continue
		}
		if !strings.Contains(string(b), "name: "+e.Name()) {
			t.Errorf("skill %s: el frontmatter debe declarar 'name: %s'", e.Name(), e.Name())
		}
	}
	want := append([]string(nil), skillNames...)
	sort.Strings(embedded)
	sort.Strings(want)
	if strings.Join(embedded, ",") != strings.Join(want, ",") {
		t.Fatalf("skills embebidas %v ≠ skillNames %v: una queda huérfana o falta", embedded, want)
	}
}

// Todo archivo embebido en core/ lo referencia el código por su ruta; si no, está en desuso.
func TestNingunArchivoEmbebidoQuedaHuerfano(t *testing.T) {
	src := goSources(t, repoRoot(t))
	fs.WalkDir(corefs.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasSuffix(p, ".go") || strings.HasPrefix(p, "skills/") {
			return nil // las skills las cubre la prueba anterior
		}
		if !strings.Contains(src, p) {
			t.Errorf("core/%s está embebido pero ningún código lo usa (huérfano)", p)
		}
		return nil
	})
}

var topCase = regexp.MustCompile(`(?m)^\tcase "([a-z-]+)"`)

// Todo comando de primer nivel está en la referencia del manual y en el README o el manual.
func TestCadaComandoEstaDocumentado(t *testing.T) {
	root := repoRoot(t)
	main, _ := os.ReadFile(filepath.Join(root, "cmd", "v-ia", "main.go"))
	manual, _ := os.ReadFile(filepath.Join(root, "docs", "manual.md"))
	ms := string(main)
	i, j := strings.Index(ms, "switch args[0] {"), strings.Index(ms, "comando desconocido")
	if i < 0 || j < i {
		t.Fatal("no encuentro el switch de comandos en main.go")
	}
	cmds := topCase.FindAllStringSubmatch(ms[i:j], -1)
	if len(cmds) < 10 {
		t.Fatalf("no pude extraer los comandos de main.go (%d)", len(cmds))
	}
	for _, m := range cmds {
		c := m[1]
		if c == "hook" || c == "version" { // internos / triviales
			continue
		}
		if !strings.Contains(string(manual), "`v-ia "+c) {
			t.Errorf("el comando %q no aparece en la referencia de docs/manual.md", c)
		}
	}
}

// Los hooks que instala settings apuntan a subcomandos que existen.
func TestHooksInstaladosApuntanASubcomandosReales(t *testing.T) {
	root := repoRoot(t)
	hook, _ := os.ReadFile(filepath.Join(root, "cmd", "v-ia", "main.go"))
	for _, sub := range []string{"pre", "prompt", "stop", "session"} {
		if !strings.Contains(string(hook), `"`+sub+`"`) {
			t.Errorf("el subcomando 'hook %s' no existe en main.go", sub)
		}
	}
}
