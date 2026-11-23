package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func skill(t *testing.T, base, name, desc string) {
	d := filepath.Join(base, name)
	os.MkdirAll(d, 0o755)
	os.WriteFile(filepath.Join(d, "SKILL.md"), []byte("---\nname: "+name+"\ndescription: "+desc+"\n---\ncuerpo"), 0o644)
}

func TestProjectBeatsGlobalAndRefreshIsIdempotent(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	skill(t, filepath.Join(home, ".claude", "skills"), "a", "global")
	skill(t, filepath.Join(home, ".claude", "skills"), "b", "solo global")
	skill(t, filepath.Join(root, ".claude", "skills"), "a", "proyecto")
	ss := Scan(home, root)
	if len(ss) != 2 || ss[0].Scope != "proyecto" || ss[0].Description != "proyecto" {
		t.Fatalf("%+v", ss)
	}
	if w, err := Refresh(home, root); err != nil || !w {
		t.Fatalf("primer refresh debe escribir: %v %v", w, err)
	}
	if w, _ := Refresh(home, root); w {
		t.Fatal("contenido idéntico no debe reescribirse")
	}
	b, _ := os.ReadFile(Path(root))
	if !strings.Contains(string(b), "| a | proyecto |") {
		t.Fatalf("%s", b)
	}
}
