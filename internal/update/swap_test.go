package update

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// failRename hace fallar los renombrados cuyo origen termina en alguno de los sufijos.
func failRename(t *testing.T, suffixes ...string) (self, tmp string) {
	t.Helper()
	d := t.TempDir()
	self, tmp = filepath.Join(d, "v-ia.exe"), filepath.Join(d, "v-ia.exe.new")
	os.WriteFile(self, []byte("VIEJO"), 0o755)
	os.WriteFile(tmp, []byte("NUEVO"), 0o755)
	old := rename
	rename = func(a, b string) error {
		for _, s := range suffixes {
			if strings.HasSuffix(a, s) {
				return errors.New("fallo simulado")
			}
		}
		return os.Rename(a, b)
	}
	t.Cleanup(func() { rename = old })
	return self, tmp
}

func TestSwapWindowsRestoresOldBinary(t *testing.T) {
	self, tmp := failRename(t, ".new")
	if swap("windows", tmp, self) == nil {
		t.Fatal("debe devolver el error")
	}
	if b, _ := os.ReadFile(self); string(b) != "VIEJO" {
		t.Fatalf("debe volver el binario anterior: %q", b)
	}
}

func TestSwapWindowsSaysWhereTheBinaryIs(t *testing.T) {
	self, tmp := failRename(t, ".new", ".old")
	err := swap("windows", tmp, self)
	if err == nil || !strings.Contains(err.Error(), self+".old") {
		t.Fatalf("el error debe decir dónde quedó el binario: %v", err)
	}
}
