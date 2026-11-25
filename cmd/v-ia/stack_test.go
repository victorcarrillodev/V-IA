package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"v-ia/internal/extras"
)

func fakeStack(t *testing.T, bins ...string) *[]string {
	t.Helper()
	var ran []string
	oldL, oldE := extras.LookPath, extras.Exec
	extras.LookPath = func(b string) (string, error) {
		for _, x := range bins {
			if x == b {
				return "/fake/" + b, nil
			}
		}
		return "", errors.New("no está")
	}
	extras.Exec = func(s extras.Step, _ io.Writer) error { ran = append(ran, s.String()); return nil }
	t.Cleanup(func() { extras.LookPath, extras.Exec = oldL, oldE })
	return &ran
}

func stdInstall(t *testing.T, args ...string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("V_IA_HOME", home)
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("CODEX_HOME", "")
	if code := run(append([]string{"install", "--yes", "--agent", "claude-code"}, args...)); code != 0 {
		t.Fatalf("salió con %d", code)
	}
}

func TestStandardIncludesCodeGraph(t *testing.T) {
	ran := fakeStack(t, "codegraph", "claude")
	stdInstall(t)
	if got := strings.Join(*ran, "\n"); !strings.Contains(got, "codegraph install") {
		t.Errorf("ejecutado: %q", got)
	}
}

func TestCodeGraphCanBeDisabled(t *testing.T) {
	for _, args := range [][]string{{"--codegraph=false"}, {"--preset", "minimal"}} {
		ran := fakeStack(t, "codegraph", "claude")
		stdInstall(t, args...)
		if len(*ran) != 0 {
			t.Errorf("%v ejecutó %v", args, *ran)
		}
	}
}

// Sin requisitos para el componente por defecto, la instalación sigue.
func TestDefaultCodeGraphMissingRequirementsDoesNotFail(t *testing.T) {
	ran := fakeStack(t, "claude")
	stdInstall(t)
	if len(*ran) != 0 {
		t.Errorf("ejecutó %v", *ran)
	}
}

// CodeGraph se registra solo en los agentes elegidos.
func TestCodeGraphFollowsSelectedAgents(t *testing.T) {
	ran := fakeStack(t, "codegraph", "claude", "opencode", "codex")
	stdInstall(t)
	if got := strings.Join(*ran, "\n"); got != "codegraph install --yes --target=claude" {
		t.Errorf("ejecutado: %q", got)
	}
}

// Al instalar con --engram, el primer install ya omite el protocolo repetido.
func TestInstallWithEngramSkipsDuplicateProtocol(t *testing.T) {
	fakeStack(t, "engram", "claude")
	home := t.TempDir()
	t.Setenv("V_IA_HOME", home)
	t.Setenv("HOME", home)
	if code := run([]string{"install", "--yes", "--agent", "claude-code", "--engram", "--codegraph=false"}); code != 0 {
		t.Fatalf("salió con %d", code)
	}
	b, err := os.ReadFile(filepath.Join(home, ".claude", "CLAUDE.md"))
	if err != nil || strings.Contains(string(b), "Qué guardar sin que te lo pidan") {
		t.Errorf("protocolo duplicado (err=%v)", err)
	}
}
