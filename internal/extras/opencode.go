package extras

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// opencodeDir es la carpeta de configuración de OpenCode.
func opencodeDir(home string) string {
	x := os.Getenv("XDG_CONFIG_HOME")
	if x == "" {
		x = filepath.Join(home, ".config")
	}
	return filepath.Join(x, "opencode")
}

// openCodeHas indica si OpenCode ya conoce un servidor MCP con ese nombre, en opencode.json
// o en opencode.jsonc (este puede llevar comentarios, así que solo se busca el nombre).
func openCodeHas(home, name string) bool {
	for _, f := range []string{"opencode.json", "opencode.jsonc"} {
		b, err := os.ReadFile(filepath.Join(opencodeDir(home), f))
		if err == nil && strings.Contains(string(b), `"`+name+`"`) {
			return true
		}
	}
	return false
}

// addOpenCodeMCP registra un servidor MCP remoto en ~/.config/opencode/opencode.json,
// conservando lo que ya hubiera. Nunca toca opencode.jsonc (suele tener comentarios del usuario).
func addOpenCodeMCP(home, name, url string) error {
	p := filepath.Join(opencodeDir(home), "opencode.json")
	cfg := map[string]any{}
	if b, err := os.ReadFile(p); err == nil {
		if err := json.Unmarshal(b, &cfg); err != nil {
			return fmt.Errorf("%s no es JSON válido; arréglalo antes (no lo toco): %w", p, err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if _, ok := cfg["$schema"]; !ok {
		cfg["$schema"] = "https://opencode.ai/config.json"
	}
	mcp, _ := cfg["mcp"].(map[string]any)
	if mcp == nil {
		mcp = map[string]any{}
	}
	mcp[name] = map[string]any{"type": "remote", "url": url, "enabled": true}
	cfg["mcp"] = mcp
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(p); err == nil {
		mode = fi.Mode().Perm() // conserva los permisos (puede llevar tokens en headers)
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), mode); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
