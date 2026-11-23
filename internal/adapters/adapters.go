// Package adapters define cómo se renderiza ODD para cada agente.
package adapters

import (
	"os"
	"os/exec"
	"path/filepath"

	corefs "v-ia/core"
	"v-ia/internal/core"
)

const Version = "0.3.0"

// Adapter describe un agente soportado.
type Adapter struct {
	Name    string
	Binary  string                     // ejecutable a buscar en PATH
	Hooks   bool                       // ¿V-IA puede hacer cumplir ODD por bloqueo (hooks)?
	Targets func(home string) []string // archivos de instrucciones globales
}

func xdgConfig(home string) string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return x
	}
	return filepath.Join(home, ".config")
}

func codexHome(home string) string {
	if c := os.Getenv("CODEX_HOME"); c != "" {
		return c
	}
	return filepath.Join(home, ".codex")
}

func one(parts ...string) []string { return []string{filepath.Join(parts...)} }

// All son los agentes con instrucciones globales conocidas (rutas verificadas contra la
// configuración oficial de cada agente). Solo Claude Code expone hooks de bloqueo;
// el resto recibe las instrucciones y `doctor` lo declara como "solo recomendación".
var All = []Adapter{
	{Name: "claude-code", Binary: "claude", Hooks: true, Targets: func(h string) []string { return one(h, ".claude", "CLAUDE.md") }},
	{Name: "opencode", Binary: "opencode", Targets: func(h string) []string { return one(xdgConfig(h), "opencode", "AGENTS.md") }},
	{Name: "codex", Binary: "codex", Targets: func(h string) []string { return one(codexHome(h), "AGENTS.md") }},
	{Name: "gemini-cli", Binary: "gemini", Targets: func(h string) []string { return one(h, ".gemini", "GEMINI.md") }},
	{Name: "kilocode", Binary: "kilo", Targets: func(h string) []string { return one(xdgConfig(h), "kilo", "AGENTS.md") }},
	{Name: "qwen-code", Binary: "qwen", Targets: func(h string) []string { return one(h, ".qwen", "QWEN.md") }},
	{Name: "windsurf", Binary: "windsurf", Targets: func(h string) []string {
		return one(h, ".codeium", "windsurf", "memories", "global_rules.md")
	}},
}

// Generic escribe AGENTS.md en el proyecto actual; sirve a cualquier agente que lo lea.
// No figura en All (la TUI y doctor son globales); se elige con --agent generic.
var Generic = Adapter{Name: "generic", Targets: func(string) []string {
	wd, err := os.Getwd()
	if err != nil {
		return nil
	}
	return one(wd, "AGENTS.md")
}}

func Find(name string) (Adapter, bool) {
	if name == Generic.Name {
		return Generic, true
	}
	for _, a := range All {
		if a.Name == name {
			return a, true
		}
	}
	return Adapter{}, false
}

func (a Adapter) Detected() bool {
	if a.Binary == "" {
		return false
	}
	_, err := exec.LookPath(a.Binary)
	return err == nil
}

// ODDBlock devuelve el bloque canónico (odd + classify + protocolo de memoria, embebidos).
func ODDBlock() (core.Block, error) {
	var content string
	for _, f := range []string{"odd/odd.md", "odd/classify.md", "memory/memory-protocol.md"} {
		b, err := corefs.FS.ReadFile(f)
		if err != nil {
			return core.Block{}, err
		}
		content += string(b) + "\n"
	}
	return core.Block{ID: "odd", Version: Version, Content: content}, nil
}

// Targets construye los core.Target de los agentes dados (los desconocidos se ignoran).
func Targets(home string, names []string) ([]core.Target, error) {
	b, err := ODDBlock()
	if err != nil {
		return nil, err
	}
	var ts []core.Target
	for _, n := range names {
		a, ok := Find(n)
		if !ok {
			continue
		}
		for _, p := range a.Targets(home) {
			ts = append(ts, core.Target{Path: p, Block: b})
		}
	}
	return ts, nil
}

// Home devuelve el HOME efectivo (V_IA_HOME permite pruebas aisladas).
func Home() (string, error) {
	if h := os.Getenv("V_IA_HOME"); h != "" {
		return h, nil
	}
	return os.UserHomeDir()
}

// SettingsPath es el settings.json global de Claude Code.
func SettingsPath(home string) string { return filepath.Join(home, ".claude", "settings.json") }
