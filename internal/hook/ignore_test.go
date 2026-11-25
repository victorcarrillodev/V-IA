package hook

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// El estado local no debe acabar en commits; la configuración de .v-ia sí puede versionarse.
func TestStateIsIgnoredByGit(t *testing.T) {
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if err := SaveState(root, State{Mode: "read"}); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"guard.json", "receipt.json", "reminded", "checks.txt"} {
		if err := os.WriteFile(filepath.Join(root, ".v-ia", f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out, err := exec.Command("git", "-C", root, "status", "--porcelain", "-uall").Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := string(out); got != "?? .v-ia/checks.txt\n" {
		t.Errorf("git ve:\n%s", got)
	}
}
