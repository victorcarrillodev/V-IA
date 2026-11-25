package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"v-ia/internal/extras"
)

// externalWriter simula un instalador externo que edita archivos que V-IA también gestiona.
func externalWriter(t *testing.T) (home string, read func(string) string) {
	t.Helper()
	fakeStack(t, "codegraph", "claude")
	home = t.TempDir()
	t.Setenv("V_IA_HOME", home)
	t.Setenv("HOME", home)
	put := func(rel, body string) {
		p := filepath.Join(home, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	put(".claude/CLAUDE.md", "mis reglas\n")
	put(".claude/settings.json", "{}\n")
	extras.Exec = func(extras.Step, io.Writer) error {
		put(".claude/CLAUDE.md", "mis reglas\nBLOQUE EXTERNO\n")
		put(".claude/settings.json", `{"externo":true}`+"\n")
		put(".claude.json", `{"mcpServers":{"codegraph":{}}}`)
		return nil
	}
	return home, func(rel string) string {
		b, _ := os.ReadFile(filepath.Join(home, rel))
		return string(b)
	}
}

func TestInstallKeepsExternalInstallerChanges(t *testing.T) {
	_, read := externalWriter(t)
	if code := run([]string{"install", "--yes", "--agent", "claude-code"}); code != 0 {
		t.Fatalf("salió con %d", code)
	}
	if got := read(".claude/CLAUDE.md"); !strings.Contains(got, "BLOQUE EXTERNO") || !strings.Contains(got, "v-ia:managed") {
		t.Errorf("CLAUDE.md perdió contenido:\n%.200s", got)
	}
	if got := read(".claude/settings.json"); !strings.Contains(got, "externo") || !strings.Contains(got, "hook") {
		t.Errorf("settings.json perdió contenido: %.200s", got)
	}
}

// restore latest vuelve al estado anterior a todo, incluido lo que escribió el instalador externo.
func TestRestoreUndoesExternalInstaller(t *testing.T) {
	home, read := externalWriter(t)
	if code := run([]string{"install", "--yes", "--agent", "claude-code"}); code != 0 {
		t.Fatalf("salió con %d", code)
	}
	if code := run([]string{"restore", "latest", "--yes"}); code != 0 {
		t.Fatalf("restore salió con %d", code)
	}
	if got := read(".claude/CLAUDE.md"); got != "mis reglas\n" {
		t.Errorf("CLAUDE.md: %.200q", got)
	}
	if got := read(".claude/settings.json"); got != "{}\n" {
		t.Errorf("settings.json: %.200q", got)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude.json")); !os.IsNotExist(err) {
		t.Error(".claude.json no existía: debe eliminarse")
	}
}
