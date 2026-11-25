package extras

import "path/filepath"

// ConfigFiles son los archivos de los agentes que los instaladores externos pueden editar.
func ConfigFiles(home string) []string {
	j := func(p ...string) string { return filepath.Join(append([]string{home}, p...)...) }
	oc, cx := opencodeDir(home), codexDir(home)
	return []string{
		j(".claude.json"), j(".claude", "settings.json"), j(".claude", "CLAUDE.md"),
		filepath.Join(oc, "opencode.json"), filepath.Join(oc, "opencode.jsonc"), filepath.Join(oc, "tui.json"), filepath.Join(oc, "AGENTS.md"),
		filepath.Join(cx, "config.toml"), filepath.Join(cx, "AGENTS.md"),
		j(".gemini", "GEMINI.md"), j(".gemini", "settings.json"), j(".gemini", "config", "mcp_config.json"),
	}
}
