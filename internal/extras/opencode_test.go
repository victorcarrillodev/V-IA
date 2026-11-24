package extras

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAddOpenCodeMCPPreservesAndIsIdempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	p := filepath.Join(home, "xdg", "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(`{"model":"x/y","mcp":{"otro":{"type":"local","command":["a"]}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := addOpenCodeMCP(home, "context7", Context7URL); err != nil {
			t.Fatal(err)
		}
	}
	var cfg struct {
		Model string                    `json:"model"`
		MCP   map[string]map[string]any `json:"mcp"`
	}
	b, _ := os.ReadFile(p)
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "x/y" || cfg.MCP["otro"] == nil || cfg.MCP["context7"]["url"] != Context7URL || len(cfg.MCP) != 2 {
		t.Fatalf("config final: %s", b)
	}
	if !openCodeHas(home, "context7") {
		t.Fatal("openCodeHas debía detectarlo")
	}
}

func TestAddOpenCodeMCPRefusesBrokenJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	p := filepath.Join(home, "xdg", "opencode", "opencode.json")
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte("{ roto"), 0o644)
	if err := addOpenCodeMCP(home, "context7", Context7URL); err == nil {
		t.Fatal("debía negarse a tocar un JSON inválido")
	}
	if b, _ := os.ReadFile(p); string(b) != "{ roto" {
		t.Fatal("no debía modificar el archivo")
	}
}
