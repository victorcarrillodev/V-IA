package review

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setup(t *testing.T, cmds string) string {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a.txt"), []byte("1"), 0o644)
	os.MkdirAll(filepath.Join(root, ".v-ia"), 0o755)
	os.WriteFile(filepath.Join(root, ".v-ia", "checks.txt"), []byte(cmds), 0o644)
	return root
}

func TestPassThenByteChangeInvalidates(t *testing.T) {
	root := setup(t, "true\n")
	r, err := Run(root, DetectCommands(root), time.Minute)
	if err != nil || r.Result != "pass" {
		t.Fatalf("esperaba pass: %v %+v", err, r)
	}
	if err := Verify(root); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "a.txt"), []byte("2"), 0o644)
	if Verify(root) == nil {
		t.Fatal("cambiar un byte debe invalidar el recibo")
	}
}

func TestFailingCheckGivesFail(t *testing.T) {
	root := setup(t, "false\n")
	r, _ := Run(root, DetectCommands(root), time.Minute)
	if r.Result != "fail" || Verify(root) == nil {
		t.Fatal("un check fallido no debe dar recibo válido")
	}
}

func TestNoChecksIsError(t *testing.T) {
	if _, err := Run(t.TempDir(), nil, time.Minute); err == nil {
		t.Fatal("sin comprobaciones no puede haber recibo")
	}
}
