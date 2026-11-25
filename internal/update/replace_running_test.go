package update

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Proceso auxiliar: la copia del binario de pruebas sigue en ejecución hasta que se cierre su entrada.
func init() {
	if os.Getenv("V_IA_TEST_WAIT") == "1" {
		io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	}
}

// update reemplaza el propio v-ia mientras se ejecuta: Windows lo trata distinto que Unix.
func TestReplaceRunningBinary(t *testing.T) {
	d := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	code, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	self, nb := filepath.Join(d, "v-ia.exe"), filepath.Join(d, "nuevo")
	os.WriteFile(self, code, 0o755)
	os.WriteFile(nb, []byte("BINARIO NUEVO"), 0o755)
	cmd := exec.Command(self)
	cmd.Env = append(os.Environ(), "V_IA_TEST_WAIT=1")
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		in.Close()
		cmd.Wait()
		time.Sleep(100 * time.Millisecond) // Windows suelta el ejecutable con retraso
	}()
	if err := Replace(self, nb); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(self); string(b) != "BINARIO NUEVO" {
		t.Fatalf("no se reemplazó: %d bytes", len(b))
	}
}
