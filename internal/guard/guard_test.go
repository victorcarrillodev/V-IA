package guard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGuardDetectaCambios(t *testing.T) {
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "a.txt"), []byte("1"), 0o644)
	os.WriteFile(filepath.Join(d, "b.txt"), []byte("1"), 0o644)
	if err := Start(d); err != nil {
		t.Fatal(err)
	}
	if ch, _ := Diff(d); len(ch) != 0 {
		t.Fatalf("sin cambios esperados: %v", ch)
	}
	os.WriteFile(filepath.Join(d, "a.txt"), []byte("2"), 0o644)
	os.Remove(filepath.Join(d, "b.txt"))
	os.WriteFile(filepath.Join(d, "c.txt"), []byte("3"), 0o644)
	ch, _ := Diff(d)
	want := []string{"+ c.txt", "- b.txt", "~ a.txt"}
	if len(ch) != 3 {
		t.Fatalf("%v", ch)
	}
	for _, w := range want {
		found := false
		for _, c := range ch {
			found = found || c == w
		}
		if !found {
			t.Fatalf("falta %q en %v", w, ch)
		}
	}
}
