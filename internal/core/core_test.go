package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var blk = Block{ID: "odd", Version: "1.0.0", Content: "hola ODD"}

func targets(p string) []Target { return []Target{{Path: p, Block: blk}} }

func TestDryRunNoEscribe(t *testing.T) {
	d := t.TempDir()
	f := filepath.Join(d, "CLAUDE.md")
	p, err := BuildPlan(targets(f), false)
	if err != nil || len(p.Changes()) != 1 {
		t.Fatalf("plan: %v %v", p, err)
	}
	if _, err := os.Stat(f); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("dry-run escribió el archivo")
	}
	if !strings.Contains(p.Actions[0].Diff(), "+hola ODD") {
		t.Fatal("diff sin contenido")
	}
}

func TestIdempotencia(t *testing.T) {
	d := t.TempDir()
	f := filepath.Join(d, "CLAUDE.md")
	os.WriteFile(f, []byte("# mis reglas\n"), 0o644)
	bk := filepath.Join(d, "bk")
	p, _ := BuildPlan(targets(f), false)
	if _, err := Apply(p, bk, time.Now()); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(f)
	p2, _ := BuildPlan(targets(f), false)
	if len(p2.Changes()) != 0 || p2.Actions[0].Status != Unchanged {
		t.Fatalf("2ª ejecución no es no-op: %v", p2.Actions[0].Status)
	}
	dir, _ := Apply(p2, bk, time.Now())
	if dir != "" {
		t.Fatal("creó backup sin cambios")
	}
	second, _ := os.ReadFile(f)
	if string(first) != string(second) || !strings.HasPrefix(string(first), "# mis reglas\n") {
		t.Fatal("contenido cambió o se pisó lo del usuario")
	}
}

func TestEdicionManualNoSePisa(t *testing.T) {
	d := t.TempDir()
	f := filepath.Join(d, "a.md")
	os.WriteFile(f, []byte(Render(blk)), 0o644)
	b, _ := os.ReadFile(f)
	os.WriteFile(f, []byte(strings.Replace(string(b), "hola ODD", "hola EDITADO", 1)), 0o644)
	nb := blk
	nb.Content, nb.Version = "nuevo", "1.1.0"
	p, _ := BuildPlan([]Target{{f, nb}}, false)
	if len(p.Conflicts()) != 1 {
		t.Fatal("no detectó conflicto")
	}
	if _, err := Apply(p, filepath.Join(d, "bk"), time.Now()); err == nil {
		t.Fatal("Apply debía fallar con conflicto")
	}
	got, _ := os.ReadFile(f)
	if !strings.Contains(string(got), "EDITADO") {
		t.Fatal("pisó la edición manual")
	}
	pf, _ := BuildPlan([]Target{{f, nb}}, true)
	if _, err := Apply(pf, filepath.Join(d, "bk"), time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreByteAByte(t *testing.T) {
	d := t.TempDir()
	existente := filepath.Join(d, "CLAUDE.md")
	nuevo := filepath.Join(d, "sub", "AGENTS.md")
	orig := []byte("regla 1\r\nsin salto final")
	os.WriteFile(existente, orig, 0o640)
	bk := filepath.Join(d, "bk")
	p, _ := BuildPlan([]Target{{existente, blk}, {nuevo, blk}}, false)
	dir, err := Apply(p, bk, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := Restore(dir); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(existente)
	if string(got) != string(orig) {
		t.Fatalf("restore no es byte a byte: %q", got)
	}
	if st, _ := os.Stat(existente); st.Mode().Perm() != 0o640 {
		t.Fatal("permisos no restaurados")
	}
	if _, err := os.Stat(nuevo); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("archivo nuevo no eliminado")
	}
	if l, _ := Latest(bk); l != dir {
		t.Fatal("Latest incorrecto")
	}
}

func TestRollbackSiFallaAMitad(t *testing.T) {
	d := t.TempDir()
	a, b := filepath.Join(d, "a.md"), filepath.Join(d, "b.md")
	os.WriteFile(a, []byte("A"), 0o644)
	os.WriteFile(b, []byte("B"), 0o644)
	n := 0
	writeHook = func(path string, data []byte, mode os.FileMode) error {
		n++
		if n == 2 {
			return errors.New("fallo simulado")
		}
		return writeAtomic(path, data, mode)
	}
	defer func() { writeHook = writeAtomic }()
	p, _ := BuildPlan([]Target{{a, blk}, {b, blk}}, false)
	if _, err := Apply(p, filepath.Join(d, "bk"), time.Now()); err == nil {
		t.Fatal("esperaba error")
	}
	ga, _ := os.ReadFile(a)
	gb, _ := os.ReadFile(b)
	if string(ga) != "A" || string(gb) != "B" {
		t.Fatalf("rollback incompleto: %q %q", ga, gb)
	}
}

func TestRemove(t *testing.T) {
	s, _ := Upsert("antes\n", true, blk, false)
	out, ok := Remove(s, "odd")
	if !ok || out != "antes\n" {
		t.Fatalf("remove: %q", out)
	}
}
