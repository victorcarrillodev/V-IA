// Package extras integra herramientas opcionales de terceros (Engram, Context7, CodeGraph).
// V-IA no reimplementa nada: planifica y ejecuta sus comandos oficiales de instalación.
// A diferencia de lifecycle, esto NO produce diff de archivos: se muestran los comandos.
package extras

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

const (
	Engram    = "engram"
	Context7  = "context7"
	CodeGraph = "codegraph"
)

// All es el orden de presentación y el significado de --all.
var All = []string{Engram, Context7, CodeGraph}

type Info struct {
	ID, Title, Desc, Warn string
}

var infos = map[string]Info{
	Engram:   {Engram, "Engram — memoria entre sesiones", "instala engram (go install o brew) si falta y ejecuta 'engram setup' en cada agente presente", ""},
	Context7: {Context7, "Context7 — documentación actual por MCP", "registra el MCP remoto de Context7 (no necesita Node; requiere claude)", ""},
	CodeGraph: {CodeGraph, "CodeGraph — grafo de código por MCP", "instala CodeGraph si falta (npm o su instalador) y lo registra en los agentes presentes",
		"su instalador también escribe instrucciones en ~/.claude/CLAUDE.md; luego ejecuta 'codegraph init -i' en cada proyecto"},
}

// Describe devuelve la ficha de un extra.
func Describe(id string) (Info, bool) { i, ok := infos[id]; return i, ok }

// Step es un comando a ejecutar.
type Step struct {
	Desc string
	Name string
	Args []string
	Env  []string // variables extra (KEY=valor)
	// Fn, si existe, se ejecuta en lugar del comando (escritura de configuración en Go).
	Fn func() error
}

func (s Step) String() string {
	if s.Fn != nil && s.Name == "" {
		return "(escritura de configuración)"
	}
	return strings.TrimSpace(s.Name + " " + strings.Join(s.Args, " "))
}

// Sustituibles en pruebas.
var (
	LookPath = exec.LookPath
	Exec     = func(s Step, out io.Writer) error {
		if s.Fn != nil {
			return s.Fn()
		}
		c := exec.Command(s.Name, s.Args...)
		c.Env = append(os.Environ(), s.Env...)
		c.Stdout, c.Stderr = out, out
		return c.Run()
	}
)

// agent describe cómo se llama cada agente principal en las herramientas externas.
type agent struct{ name, bin, engram, codegraph string }

var primary = []agent{
	{"claude-code", "claude", "claude-code", "claude"},
	{"opencode", "opencode", "opencode", "opencode"},
	{"codex", "codex", "codex", "codex"},
	{"antigravity", "agy", "antigravity-cli", "antigravity"},
}

func codexDir(home string) string {
	if c := os.Getenv("CODEX_HOME"); c != "" {
		return c
	}
	return filepath.Join(home, ".codex")
}

func (a agent) present(home string) bool {
	if have(a.bin) {
		return true
	}
	var dir string
	switch a.name {
	case "opencode":
		dir = opencodeDir(home)
	case "codex":
		dir = codexDir(home)
	case "antigravity":
		dir = filepath.Join(home, ".gemini", "antigravity-cli")
	default:
		return false
	}
	st, err := os.Stat(dir)
	return err == nil && st.IsDir()
}

// has indica si el agente ya conoce el servidor MCP.
func (a agent) has(home, name string) bool {
	switch a.name {
	case "opencode":
		return openCodeHas(home, name)
	case "codex":
		b, err := os.ReadFile(filepath.Join(codexDir(home), "config.toml"))
		return err == nil && strings.Contains(string(b), "[mcp_servers."+name+"]")
	case "antigravity":
		b, err := os.ReadFile(filepath.Join(home, ".gemini", "config", "mcp_config.json"))
		return err == nil && strings.Contains(string(b), `"`+name+`"`)
	}
	return registered(home, name)
}

// HasServer indica si un agente principal ya tiene configurado un servidor MCP.
func HasServer(home, agentName, server string) bool {
	for _, a := range primary {
		if a.name == agentName {
			return a.has(home, server)
		}
	}
	return false
}

// targets son los agentes presentes, limitados a only si no es nil. Sin ninguno presente
// ni elección, Claude Code.
func targets(home string, only []string) []agent {
	var out []agent
	for _, a := range primary {
		if (only == nil || slices.Contains(only, a.name)) && a.present(home) {
			out = append(out, a)
		}
	}
	if len(out) == 0 && (only == nil || slices.Contains(only, "claude-code")) {
		out = primary[:1]
	}
	return out
}

func allHave(home, name string, as []agent) bool {
	for _, a := range as {
		if !a.has(home, name) {
			return false
		}
	}
	return true
}

// ocPresent indica si OpenCode está instalado: entonces los MCP también se registran allí.
func ocPresent() bool { return have("opencode") }

func have(bin string) bool { _, err := LookPath(bin); return err == nil }

// Versiones y orígenes fijados (cadena de suministro): se actualizan a propósito, no con "latest".
const (
	Context7URL     = "https://mcp.context7.com/mcp"
	CodeGraphPkg    = "@colbymchenry/codegraph@1.6.4"
	CodeGraphScript = "https://raw.githubusercontent.com/colbymchenry/codegraph/main/install.sh"
	EngramPkg       = "github.com/Gentleman-Programming/engram/v3/cmd/engram@v3.7.2"
	EngramBrew      = "gentleman-programming/tap/engram"
)

// binDir es donde V-IA deja los binarios que instala (sin sudo).
func binDir() string {
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".local", "bin")
}

// registered detecta si Claude Code ya conoce un servidor MCP de alcance usuario, leyendo
// "mcpServers" de ~/.claude.json (más rápido y seguro que 'claude mcp list', que conecta con
// cada servidor, y sin falsos positivos por rutas o historial).
func registered(home, name string) bool {
	b, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err != nil {
		return false
	}
	var cfg struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if json.Unmarshal(b, &cfg) != nil {
		return false
	}
	_, ok := cfg.MCPServers[name]
	return ok
}

// Installed indica si el extra ya está listo (binario presente y servidor MCP registrado).
func Installed(home, id string) bool { return installed(home, id, targets(home, nil)) }

func installed(home, id string, as []agent) bool {
	switch id {
	case Engram:
		return have("engram") && allHave(home, "engram", as)
	case Context7:
		return (!have("claude") || registered(home, "context7")) && (!ocPresent() || openCodeHas(home, "context7")) && (have("claude") || ocPresent())
	case CodeGraph:
		return have("codegraph") && allHave(home, "codegraph", as)
	}
	return false
}

// MissingError indica requisitos ausentes. Plan lo devuelve junto con los pasos, para que
// --dry-run pueda mostrar qué haría y avisar de lo que falta.
type MissingError struct{ Items []string }

func (e *MissingError) Error() string { return "faltan requisitos: " + strings.Join(e.Items, "; ") }

// IsMissing indica si err es un MissingError.
func IsMissing(err error) bool { var m *MissingError; return errors.As(err, &m) }

// Plan devuelve los pasos pendientes y avisos. Si falta un requisito devuelve además un
// *MissingError (los pasos siguen informados, pero no deben ejecutarse).
func Plan(home string, ids []string) (steps []Step, notes []string, err error) {
	return PlanFor(home, ids, nil)
}

// PlanFor limita Engram y CodeGraph a los agentes dados (nil = todos los presentes).
func PlanFor(home string, ids, only []string) (steps []Step, notes []string, err error) {
	as := targets(home, only)
	var missing []string
	need := func(id string, bins ...string) {
		for _, b := range bins {
			if !have(b) {
				missing = append(missing, fmt.Sprintf("%s necesita '%s' en PATH", id, b))
			}
		}
	}
	for _, id := range ids {
		if _, ok := infos[id]; !ok {
			return nil, nil, fmt.Errorf("componente desconocido: %s (disponibles: %s)", id, strings.Join(All, ", "))
		}
		if installed(home, id, as) {
			notes = append(notes, id+": ya instalado (sin cambios)")
			continue
		}
		switch id {
		case Engram:
			if !have("engram") {
				switch {
				case have("go"):
					steps = append(steps, Step{"instalar Engram con go install (en ~/.local/bin)", "go", []string{"install", EngramPkg}, []string{"GOBIN=" + binDir()}, nil})
				case have("brew"):
					steps = append(steps, Step{"instalar Engram con Homebrew", "brew", []string{"install", EngramBrew}, nil, nil})
				default:
					missing = append(missing, "engram no está y no hay 'go' ni 'brew' para instalarlo (binarios: https://github.com/Gentleman-Programming/engram/releases)")
				}
			}
			for _, a := range as {
				if a.name == "codex" && !have(a.bin) {
					notes = append(notes, "engram: codex no está en PATH; añádelo y ejecuta 'engram setup codex'")
					continue
				}
				if !a.has(home, "engram") {
					steps = append(steps, Step{Desc: "configurar Engram para " + a.name, Name: "engram", Args: []string{"setup", a.engram}})
				}
			}
		case Context7:
			if !have("claude") && !ocPresent() {
				need(id, "claude")
			}
			if (have("claude") || !ocPresent()) && !registered(home, "context7") {
				steps = append(steps, Step{Desc: "registrar Context7 en Claude Code (MCP remoto, alcance de usuario)", Name: "claude",
					Args: []string{"mcp", "add", "--scope", "user", "--transport", "http", "context7", Context7URL}})
			}
			if ocPresent() && !openCodeHas(home, "context7") {
				steps = append(steps, Step{Desc: "registrar Context7 en OpenCode (~/.config/opencode/opencode.json)", Fn: func() error { return addOpenCodeMCP(home, "context7", Context7URL) }})
			}
		case CodeGraph:
			if len(as) == 0 {
				notes = append(notes, "codegraph: ninguno de los agentes elegidos lo admite")
				continue
			}
			if len(as) == 1 && as[0].name == "claude-code" {
				need(id, "claude")
			}
			if !have("codegraph") {
				switch {
				case have("npm"):
					steps = append(steps, Step{"instalar CodeGraph con npm (global)", "npm", []string{"install", "-g", CodeGraphPkg}, nil, nil})
				case have("curl"):
					steps = append(steps, Step{"instalar CodeGraph con su instalador oficial (descarga y ejecuta un script remoto)", "sh",
						[]string{"-c", "curl -fsSL " + CodeGraphScript + " | sh"}, nil, nil})
				default:
					missing = append(missing, "codegraph no está y no hay 'npm' ni 'curl' para instalarlo")
				}
			}
			var ids, names []string
			for _, a := range as {
				ids, names = append(ids, a.codegraph), append(names, a.name)
			}
			steps = append(steps, Step{"registrar CodeGraph en " + strings.Join(names, ", "), "codegraph", []string{"install", "--yes", "--target=" + strings.Join(ids, ",")}, nil, nil})
			notes = append(notes, "codegraph: "+infos[id].Warn)
		}
	}
	if len(missing) > 0 {
		return steps, notes, &MissingError{Items: missing}
	}
	return steps, notes, nil
}

// Run ejecuta los pasos en orden y se detiene en el primer error.
func Run(steps []Step, out io.Writer) error {
	// lo instalado en ~/.local/bin debe resolverse en los pasos siguientes
	if p := os.Getenv("PATH"); !strings.Contains(":"+p+":", ":"+binDir()+":") {
		os.Setenv("PATH", binDir()+":"+p)
	}
	for _, s := range steps {
		fmt.Fprintf(out, "→ %s: %s\n", s.Desc, s)
		if err := Exec(s, out); err != nil {
			return fmt.Errorf("%s: %w", s, err)
		}
	}
	return nil
}

// Item es el estado de un extra opcional para doctor.
type Item struct {
	ID, Msg string
	OK      bool
}

// Status resume los tres opcionales.
func Status(home string) []Item {
	var out []Item
	for _, id := range []string{Engram, Context7, CodeGraph} {
		ok := Installed(home, id)
		msg := id + ": instalado"
		if !ok {
			msg = id + ": no instalado (opcional) → v-ia deps"
		}
		out = append(out, Item{id, msg, ok})
	}
	return out
}
