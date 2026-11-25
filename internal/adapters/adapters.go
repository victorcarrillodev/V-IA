// Package adapters define cómo se renderiza ODD para cada agente.
package adapters

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	corefs "v-ia/core"
	"v-ia/internal/core"
)

const Version = "0.3.0"

// Adapter describe un agente soportado.
type Adapter struct {
	Name   string
	Binary string // ejecutable a buscar en PATH
	Hooks  bool   // ¿V-IA puede hacer cumplir ODD por bloqueo (hooks)?
	// HooksVerified indica que el bloqueo se probó con el agente real.
	HooksVerified bool
	Targets       func(home string) []string // archivos de instrucciones globales
	// SkillsDir es la carpeta global de skills del agente ("" = el agente no las admite aquí).
	SkillsDir func(home string) string
	// Verified indica que la ruta de skills se probó con el agente real.
	SkillsVerified bool
	// AgentsDir es la carpeta de subagentes en archivos ("" = no los admite); AgentStyle elige el frontmatter.
	AgentsDir  func(home string) string
	AgentStyle string // claude | opencode | cursor
	// OwnFile: el agente carga un archivo de instrucciones propio (con frontmatter) en vez de un bloque
	// dentro de uno compartido: V-IA lo escribe entero como archivo gestionado.
	OwnFile func(home string) (path, frontmatter string)
	// DetectDirs: carpetas cuya existencia indica que el agente está instalado (sin binario en PATH).
	DetectDirs func(home string) []string
}

// XDGConfig es $XDG_CONFIG_HOME o ~/.config.
func XDGConfig(home string) string { return xdgConfig(home) }

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

func copilotHome(home string) string {
	if c := os.Getenv("COPILOT_HOME"); c != "" {
		return c
	}
	return filepath.Join(home, ".copilot")
}

func one(parts ...string) []string { return []string{filepath.Join(parts...)} }

// All son los agentes con instrucciones globales conocidas (rutas verificadas contra la
// configuración oficial de cada agente). Solo Claude Code expone hooks de bloqueo;
// el resto recibe las instrucciones y `doctor` lo declara como "solo recomendación".
var All = []Adapter{
	{Name: "claude-code", Binary: "claude", Hooks: true, HooksVerified: true, Targets: func(h string) []string { return one(h, ".claude", "CLAUDE.md") },
		SkillsDir: func(h string) string { return filepath.Join(h, ".claude", "skills") }, SkillsVerified: true,
		AgentsDir: func(h string) string { return filepath.Join(h, ".claude", "agents") }, AgentStyle: "claude"},
	{Name: "opencode", Binary: "opencode", Hooks: true, HooksVerified: true, Targets: func(h string) []string { return one(xdgConfig(h), "opencode", "AGENTS.md") },
		SkillsDir: func(h string) string { return filepath.Join(xdgConfig(h), "opencode", "skills") },
		AgentsDir: func(h string) string { return filepath.Join(xdgConfig(h), "opencode", "agents") }, AgentStyle: "opencode"},
	{Name: "codex", Binary: "codex", Hooks: true, Targets: func(h string) []string { return one(codexHome(h), "AGENTS.md") },
		SkillsDir: func(h string) string { return filepath.Join(codexHome(h), "skills") }},
	{Name: "gemini-cli", Binary: "gemini", Hooks: true, Targets: func(h string) []string { return one(h, ".gemini", "GEMINI.md") },
		SkillsDir: func(h string) string { return filepath.Join(h, ".gemini", "skills") }},
	{Name: "kilocode", Binary: "kilo", Targets: func(h string) []string { return one(xdgConfig(h), "kilo", "AGENTS.md") },
		SkillsDir: func(h string) string { return filepath.Join(xdgConfig(h), "kilo", "skills") }},
	{Name: "qwen-code", Binary: "qwen", Targets: func(h string) []string { return one(h, ".qwen", "QWEN.md") },
		SkillsDir: func(h string) string { return filepath.Join(h, ".qwen", "skills") }},
	{Name: "windsurf", Binary: "windsurf", Targets: func(h string) []string {
		return one(h, ".codeium", "windsurf", "memories", "global_rules.md")
	}, SkillsDir: func(h string) string { return filepath.Join(h, ".codeium", "windsurf", "skills") }},
	{Name: "cursor", Binary: "cursor", Hooks: true, Targets: func(string) []string { return nil },
		OwnFile: func(h string) (string, string) {
			return filepath.Join(h, ".cursor", "rules", "v-ia.mdc"), "---\ndescription: V-IA (ODD): método de trabajo del agente\nalwaysApply: true\n---\n"
		},
		DetectDirs: func(h string) []string { return []string{filepath.Join(h, ".cursor")} },
		SkillsDir:  func(h string) string { return filepath.Join(h, ".cursor", "skills") },
		AgentsDir:  func(h string) string { return filepath.Join(h, ".cursor", "agents") }, AgentStyle: "cursor"},
	{Name: "kiro-ide", Binary: "kiro", Targets: func(string) []string { return nil },
		OwnFile: func(h string) (string, string) {
			return filepath.Join(h, ".kiro", "steering", "v-ia.md"), "---\ninclusion: always\n---\n"
		},
		DetectDirs: func(h string) []string { return []string{filepath.Join(h, ".kiro")} },
		SkillsDir:  func(h string) string { return filepath.Join(h, ".kiro", "skills") }},
	{Name: "vscode-copilot", Binary: "code", Targets: func(string) []string { return nil },
		OwnFile: func(h string) (string, string) {
			return filepath.Join(userAppDir(h, "Code"), "prompts", "v-ia.instructions.md"), "---\nname: V-IA\ndescription: ODD, método de trabajo del agente\napplyTo: \"**\"\n---\n"
		},
		SkillsDir: func(h string) string { return filepath.Join(h, ".copilot", "skills") }},
	{Name: "trae-ide", Targets: func(h string) []string { return one(userAppDir(h, "Trae"), "user_rules.md") },
		DetectDirs: func(h string) []string { return []string{filepath.Join(h, ".trae")} },
		SkillsDir:  func(h string) string { return filepath.Join(h, ".trae", "skills") }},
	{Name: "hermes", Binary: "hermes", Targets: func(h string) []string { return one(hermesHome(h), "SOUL.md") },
		SkillsDir: func(h string) string { return filepath.Join(hermesHome(h), "skills") }},
	{Name: "kimi", Binary: "kimi", Targets: func(h string) []string {
		if kimiLegacy(h) {
			return one(h, ".kimi", "KIMI.md")
		}
		return one(h, ".kimi-code", "AGENTS.md")
	},
		DetectDirs: func(h string) []string { return []string{filepath.Join(h, ".kimi-code"), filepath.Join(h, ".kimi")} },
		SkillsDir: func(h string) string {
			if kimiLegacy(h) {
				return filepath.Join(xdgConfig(h), "agents", "skills")
			}
			return filepath.Join(h, ".kimi-code", "skills")
		}},
	{Name: "antigravity", Binary: "agy", Targets: func(string) []string { return nil },
		OwnFile: func(h string) (string, string) {
			return filepath.Join(h, ".gemini", "config", "rules", "v-ia-odd.md"), "---\ntrigger: always_on\n---\n"
		},
		DetectDirs: func(h string) []string {
			return []string{filepath.Join(h, ".gemini", "antigravity-cli"), filepath.Join(h, ".gemini", "antigravity"), filepath.Join(h, ".gemini", "config")}
		},
		SkillsDir: func(h string) string { return filepath.Join(h, ".gemini", "config", "skills") }},
	{Name: "pi", Binary: "pi", Targets: func(h string) []string { return one(piAgentDir(h), "APPEND_SYSTEM.md") },
		DetectDirs: func(h string) []string { return []string{filepath.Join(h, ".pi")} }},
	{Name: "openclaw", Binary: "openclaw", Targets: func(h string) []string { return one(h, "AGENTS.md") },
		DetectDirs: func(h string) []string { return []string{filepath.Join(h, ".openclaw")} },
		SkillsDir:  func(h string) string { return filepath.Join(h, ".openclaw", "skills") }},
	{Name: "copilot-cli", Binary: "copilot", Targets: func(h string) []string { return one(copilotHome(h), "copilot-instructions.md") },
		SkillsDir: func(h string) string { return filepath.Join(copilotHome(h), "skills") }},
}

// Primary son los agentes que se instalan por defecto y que doctor informa siempre.
var Primary = []string{"claude-code", "opencode", "codex", "antigravity"}

// IsPrimary indica si el agente está en Primary.
func IsPrimary(name string) bool { return slices.Contains(Primary, name) }

// Generic escribe AGENTS.md en el proyecto actual; sirve a cualquier agente que lo lea.
// No figura en All (la TUI y doctor son globales); se elige con --agent generic.
var Generic = Adapter{Name: "generic", Targets: func(string) []string {
	wd, err := os.Getwd()
	if err != nil {
		return nil
	}
	return one(wd, "AGENTS.md")
}}

// WorkspaceAgents son los agentes cuya disposición ~/.carpeta/ es la misma dentro de un proyecto,
// por lo que `install --scope workspace` puede reutilizar su adaptador con el proyecto como base.
var WorkspaceAgents = []string{"claude-code", "gemini-cli", "qwen-code", "kimi", "generic"}

// SupportsWorkspace indica si el agente admite --scope workspace.
func SupportsWorkspace(name string) bool {
	for _, n := range WorkspaceAgents {
		if n == name {
			return true
		}
	}
	return false
}

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
	if a.Binary != "" {
		if _, err := exec.LookPath(a.Binary); err == nil {
			return true
		}
	}
	if a.DetectDirs != nil {
		// sin HOME resoluble no se puede mirar
		if h, err := Home(); err == nil {
			for _, d := range a.DetectDirs(h) {
				if fi, err := os.Stat(d); err == nil && fi.IsDir() {
					return true
				}
			}
		}
	}
	return false
}

func isDir(p string) bool { st, err := os.Stat(p); return err == nil && st.IsDir() }

// kimiLegacy: la raíz actual (kimi-code v0.11+) es ~/.kimi-code; solo si no existe y sí existe
// la heredada ~/.kimi se usa su disposición (KIMI.md y skills compartidos en ~/.config/agents).
func kimiLegacy(home string) bool {
	return !isDir(filepath.Join(home, ".kimi-code")) && isDir(filepath.Join(home, ".kimi"))
}

// piAgentDir respeta PI_CODING_AGENT_DIR como el propio Pi: "~/..." se expande contra home;
// una ruta absoluta solo se acepta si home es el HOME real (una prueba o sandbox con otro home
// nunca se redirige fuera de él). Sin override: ~/.pi/agent.
func piAgentDir(home string) string {
	if o := strings.TrimSpace(os.Getenv("PI_CODING_AGENT_DIR")); o != "" {
		if o == "~" {
			return home
		}
		if rest, ok := strings.CutPrefix(o, "~/"); ok {
			return filepath.Join(home, rest)
		}
		if real, err := os.UserHomeDir(); err == nil && filepath.Clean(home) == filepath.Clean(real) && filepath.IsAbs(o) {
			return filepath.Clean(o)
		}
	}
	return filepath.Join(home, ".pi", "agent")
}

// userAppDir es la carpeta de usuario de una aplicación de escritorio (VS Code, Trae…): ~/.config/<App>/User
// en Linux (respeta XDG_CONFIG_HOME), ~/Library/Application Support/<App>/User en macOS y %APPDATA%\<App>\User en Windows.
func userAppDir(home, app string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", app, "User")
	case "windows":
		if a := os.Getenv("APPDATA"); a != "" {
			return filepath.Join(a, app, "User")
		}
		return filepath.Join(home, "AppData", "Roaming", app, "User")
	}
	return filepath.Join(xdgConfig(home), app, "User")
}

func hermesHome(home string) string {
	if h := os.Getenv("HERMES_HOME"); h != "" {
		return h
	}
	return filepath.Join(home, ".hermes")
}

// ODDBlock devuelve el bloque canónico. enforced indica si el agente tiene hooks de V-IA:
// el texto no debe prometer bloqueos ni skills a un agente que no los tiene.
func ODDBlock(enforced bool) (core.Block, error) {
	tail := "odd/advisory.md"
	if enforced {
		tail = "odd/enforced.md"
	}
	var content string
	for _, f := range []string{"odd/odd.md", "odd/delegate.md", tail, "odd/classify.md", "memory/memory-protocol.md"} {
		b, err := corefs.FS.ReadFile(f)
		if err != nil {
			return core.Block{}, err
		}
		content += string(b) + "\n"
	}
	return core.Block{ID: "odd", Version: Version, Content: content}, nil
}

// Targets construye los core.Target de los agentes dados (los desconocidos se ignoran).
// Personas disponibles; "" y "neutral" no añaden nada.
var Personas = []string{"neutral", "mentor", "direct"}

var personaFiles = map[string]string{"mentor": "personas/mentor.md", "direct": "personas/direct.md"}

// PersonaBlock devuelve el bloque de una persona (ok=false para neutral o desconocida).
func PersonaBlock(name string) (core.Block, bool) {
	if name == "" || name == "neutral" {
		return core.Block{}, false
	}
	f, known := personaFiles[name]
	if !known {
		return core.Block{}, false
	}
	b, err := corefs.FS.ReadFile(f)
	if err != nil {
		return core.Block{}, false
	}
	return core.Block{ID: "persona", Version: Version, Content: string(b)}, true
}

func Targets(home string, names []string, engramPlanned ...bool) ([]core.Target, error) {
	planned := len(engramPlanned) > 0 && engramPlanned[0]
	var ts []core.Target
	for _, n := range names {
		a, ok := Find(n)
		if !ok {
			continue
		}
		b, err := a.Block(home, planned)
		if err != nil {
			return nil, err
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

// HookConfigPath es el archivo donde un agente (distinto de Claude Code) guarda sus hooks;
// "" si V-IA no sabe enganchar hooks en él.
func HookConfigPath(home, agent string) string {
	switch agent {
	case "gemini-cli":
		return filepath.Join(home, ".gemini", "settings.json")
	case "codex":
		return filepath.Join(codexHome(home), "hooks.json")
	case "cursor":
		return filepath.Join(home, ".cursor", "hooks.json")
	}
	return ""
}

// SettingsPath es el settings.json global de Claude Code.
func SettingsPath(home string) string { return filepath.Join(home, ".claude", "settings.json") }
