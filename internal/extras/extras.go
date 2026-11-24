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
	Engram:   {Engram, "Engram — memoria entre sesiones", "ejecuta 'engram setup claude-code' (requiere engram)", ""},
	Context7: {Context7, "Context7 — documentación actual por MCP", "registra el servidor MCP @upstash/context7-mcp (requiere claude y Node/npx)", ""},
	CodeGraph: {CodeGraph, "CodeGraph — grafo de código por MCP", "instala @colbymchenry/codegraph y lo registra en Claude Code (requiere claude y npm)",
		"su instalador también escribe instrucciones en ~/.claude/CLAUDE.md; luego ejecuta 'codegraph init -i' en cada proyecto"},
}

// Describe devuelve la ficha de un extra.
func Describe(id string) (Info, bool) { i, ok := infos[id]; return i, ok }

// Step es un comando a ejecutar.
type Step struct {
	Desc string
	Name string
	Args []string
}

func (s Step) String() string { return strings.TrimSpace(s.Name + " " + strings.Join(s.Args, " ")) }

// Sustituibles en pruebas.
var (
	LookPath = exec.LookPath
	Exec     = func(s Step, out io.Writer) error {
		c := exec.Command(s.Name, s.Args...)
		c.Stdout, c.Stderr = out, out
		return c.Run()
	}
)

func have(bin string) bool { _, err := LookPath(bin); return err == nil }

// Versiones fijadas (cadena de suministro): se actualizan a propósito, no con "latest".
const (
	Context7Pkg  = "@upstash/context7-mcp@4.7.3"
	CodeGraphPkg = "@colbymchenry/codegraph@1.6.4"
)

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

// Installed indica si el extra ya está listo. Engram no se puede saber: siempre false.
func Installed(home, id string) bool {
	switch id {
	case Context7:
		return registered(home, "context7")
	case CodeGraph:
		return have("codegraph") && registered(home, "codegraph")
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
		if Installed(home, id) {
			notes = append(notes, id+": ya instalado (sin cambios)")
			continue
		}
		switch id {
		case Engram:
			need(id, "engram")
			steps = append(steps, Step{"configurar Engram para Claude Code", "engram", []string{"setup", "claude-code"}})
		case Context7:
			need(id, "claude", "npx")
			steps = append(steps, Step{"registrar Context7 (MCP, alcance de usuario)", "claude",
				[]string{"mcp", "add", "--scope", "user", "context7", "--", "npx", "-y", Context7Pkg}})
		case CodeGraph:
			need(id, "claude")
			if !have("codegraph") {
				need(id, "npm")
				steps = append(steps, Step{"instalar CodeGraph (global)", "npm", []string{"install", "-g", CodeGraphPkg}})
			}
			steps = append(steps, Step{"registrar CodeGraph en Claude Code", "codegraph", []string{"install", "--yes", "--target=claude"}})
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

// Status resume Context7 y CodeGraph (Engram ya lo informa doctor).
func Status(home string) []Item {
	var out []Item
	for _, id := range []string{Context7, CodeGraph} {
		ok := Installed(home, id)
		msg := id + ": instalado"
		if !ok {
			msg = id + ": no instalado (opcional) → v-ia install --" + id
		}
		out = append(out, Item{id, msg, ok})
	}
	return out
}
