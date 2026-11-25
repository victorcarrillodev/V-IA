package adapters

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAntigravityDetectsCLIBinary(t *testing.T) {
	bin := t.TempDir()
	name := "agy"
	if runtime.GOOS == "windows" {
		name = "agy.exe" // LookPath solo encuentra ejecutables con extensión
	}
	if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("V_IA_HOME", t.TempDir())
	a, _ := Find("antigravity")
	if !a.Detected() {
		t.Fatal("agy en PATH debe detectar antigravity")
	}
}

func TestAntigravityDetectsDirs(t *testing.T) {
	// antigravity-cli es nuevo; antigravity y config ya se detectaban.
	for _, dir := range []string{"antigravity-cli", "antigravity", "config"} {
		home := t.TempDir()
		if err := os.MkdirAll(filepath.Join(home, ".gemini", dir), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", t.TempDir())
		t.Setenv("V_IA_HOME", home)
		a, _ := Find("antigravity")
		if !a.Detected() {
			t.Errorf("~/.gemini/%s debe detectar antigravity", dir)
		}
	}
}
