package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func installInto(t *testing.T, extra ...string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("V_IA_HOME", home)
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("CODEX_HOME", "")
	args := append([]string{"install", "--yes", "--preset", "minimal"}, extra...)
	if code := run(args); code != 0 {
		t.Fatalf("install salió con %d", code)
	}
	return home
}

var primaryFiles = []string{
	".claude/CLAUDE.md",
	".config/opencode/AGENTS.md",
	".codex/AGENTS.md",
	".gemini/config/rules/v-ia-odd.md",
}

func exists(home, rel string) bool {
	_, err := os.Stat(filepath.Join(home, rel))
	return err == nil
}

func TestInstallDefaultsToPrimaryAgents(t *testing.T) {
	home := installInto(t)
	for _, f := range primaryFiles {
		if !exists(home, f) {
			t.Errorf("falta %s", f)
		}
	}
}

func TestInstallExplicitAgentUnchanged(t *testing.T) {
	home := installInto(t, "--agent", "opencode")
	for _, f := range primaryFiles {
		if want := f == ".config/opencode/AGENTS.md"; exists(home, f) != want {
			t.Errorf("%s: existe=%v, quería %v", f, !want, want)
		}
	}
}

// sync y uninstall sin --agent actúan sobre lo instalado, no añaden agentes.
func TestSyncWithoutAgentKeepsRecordedAgents(t *testing.T) {
	home := installInto(t, "--agent", "opencode")
	if code := run([]string{"sync", "--yes"}); code != 0 {
		t.Fatalf("sync salió con %d", code)
	}
	for _, f := range primaryFiles {
		if want := f == ".config/opencode/AGENTS.md"; exists(home, f) != want {
			t.Errorf("%s: existe=%v, quería %v", f, !want, want)
		}
	}
}

func TestUninstallWithoutAgentRemovesRecordedAgents(t *testing.T) {
	home := installInto(t, "--agent", "gemini-cli")
	if code := run([]string{"uninstall", "--yes"}); code != 0 {
		t.Fatalf("uninstall salió con %d", code)
	}
	if exists(home, ".v-ia/install.json") {
		t.Error("el registro debe desaparecer")
	}
	if b, _ := os.ReadFile(filepath.Join(home, ".gemini/GEMINI.md")); strings.Contains(string(b), "v-ia:managed") {
		t.Error("el bloque de gemini-cli debe retirarse")
	}
}
