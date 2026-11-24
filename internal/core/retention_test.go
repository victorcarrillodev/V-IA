package core

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestPruneKeepsRecentAndPinned(t *testing.T) {
	root := t.TempDir()
	for i := 1; i <= 8; i++ {
		if err := os.MkdirAll(filepath.Join(root, fmt.Sprintf("2026010%dT000000Z", i)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	old := filepath.Join(root, "20260101T000000Z")
	if err := Pin(old); err != nil {
		t.Fatal(err)
	}
	removed, err := Prune(root, 5)
	if err != nil || len(removed) != 2 {
		t.Fatalf("removed=%v err=%v", removed, err)
	}
	l, _ := ListBackups(root)
	if len(l) != 6 || l[0] != old {
		t.Fatalf("quedan %v", l)
	}
	if err := Unpin(old); err != nil || IsPinned(old) {
		t.Fatal("unpin falló")
	}
	if removed, _ = Prune(root, 5); len(removed) != 1 || removed[0] != old {
		t.Fatalf("tras unpin debía borrar el antiguo: %v", removed)
	}
}
