package extras

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func commands(t *testing.T, home string, ids ...string) string {
	t.Helper()
	steps, _, err := Plan(home, ids)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, s := range steps {
		out = append(out, s.String())
	}
	return strings.Join(out, "\n")
}

func TestEngramForPrimaryAgents(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("CODEX_HOME", "")
	fakePath(t, "engram", "claude", "opencode", "codex", "agy")
	got := commands(t, t.TempDir(), Engram)
	for _, a := range []string{"claude-code", "opencode", "codex", "antigravity-cli"} {
		if !strings.Contains(got, "engram setup "+a) {
			t.Errorf("falta %s en:\n%s", a, got)
		}
	}
}

func TestCodeGraphForPrimaryAgents(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("CODEX_HOME", "")
	fakePath(t, "codegraph", "claude", "opencode", "codex", "agy")
	got := commands(t, t.TempDir(), CodeGraph)
	if want := "codegraph install --yes --target=claude,opencode,codex,antigravity"; got != want {
		t.Errorf("got %q", got)
	}
}

// Ya registrado en los cuatro: no se repite nada.
func TestRegisteredEverywhereIsInstalled(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("CODEX_HOME", "")
	fakePath(t, "codegraph", "claude", "opencode", "codex", "agy")
	home := t.TempDir()
	for f, body := range map[string]string{
		".claude.json":                   `{"mcpServers":{"codegraph":{}}}`,
		".config/opencode/opencode.json": `{"mcp":{"codegraph":{}}}`,
		".codex/config.toml":             "[mcp_servers.codegraph]\n",
		".gemini/config/mcp_config.json": `{"mcpServers":{"codegraph":{}}}`,
	} {
		p := filepath.Join(home, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if !Installed(home, CodeGraph) {
		t.Error("debe constar como instalado")
	}
}

// Sin el ejecutable de Codex, 'engram setup codex' queda incompleto: se avisa y no se ejecuta.
func TestEngramSkipsCodexWithoutBinary(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("CODEX_HOME", "")
	fakePath(t, "engram", "claude")
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	steps, notes, err := Plan(home, []string{Engram})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range steps {
		if strings.Contains(s.String(), "setup codex") {
			t.Errorf("no debe ejecutar %s", s)
		}
	}
	if !strings.Contains(strings.Join(notes, "\n"), "codex") {
		t.Errorf("falta el aviso: %v", notes)
	}
}
