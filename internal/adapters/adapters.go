// Package adapters define cómo se renderiza ODD para cada agente.
package adapters

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	corefs "v-ia/core"
	"v-ia/internal/core"
	"v-ia/internal/settings"
)

const Version = "0.2.0"

// Adapter describe un agente soportado.
type Adapter struct {
	Name    string
	Binary  string                     // ejecutable a buscar en PATH
	Targets func(home string) []string // archivos de instrucciones globales
}

var All = []Adapter{
	{Name: "claude-code", Binary: "claude", Targets: func(h string) []string {
		return []string{filepath.Join(h, ".claude", "CLAUDE.md")}
	}},
}

// Generic escribe AGENTS.md en el proyecto actual; sirve a cualquier agente que lo lea.
// No figura en All (la TUI y doctor son globales); se elige con --agent generic.
var Generic = Adapter{Name: "generic", Targets: func(string) []string {
	wd, err := os.Getwd()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(wd, "AGENTS.md")}
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
	_, err := exec.LookPath(a.Binary)
	return err == nil
}

// ODDBlock devuelve el bloque canónico (odd.md + classify.md embebidos).
func ODDBlock() (core.Block, error) {
	odd, err := corefs.FS.ReadFile("odd/odd.md")
	if err != nil {
		return core.Block{}, err
	}
	cl, err := corefs.FS.ReadFile("odd/classify.md")
	if err != nil {
		return core.Block{}, err
	}
	return core.Block{ID: "odd", Version: Version, Content: string(odd) + "\n" + string(cl)}, nil
}

// Plan construye los Target de los agentes dados.
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

// exeForHooks devuelve el comando con el que los hooks invocarán a v-ia.
func exeForHooks() string {
	if exe, err := os.Executable(); err == nil && filepath.Base(exe) == "v-ia" {
		return exe
	}
	return "v-ia"
}

// SettingsAction planifica la instalación de los hooks de V-IA en settings.json.
func SettingsAction(home string) (core.Action, error) {
	p := SettingsPath(home)
	data, err := os.ReadFile(p)
	exists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return core.Action{}, err
	}
	nw, err := settings.Merge(string(data), exeForHooks())
	if err != nil {
		return core.Action{}, err
	}
	st := core.Update
	switch {
	case !exists:
		st = core.Create
	case nw == string(data):
		st = core.Unchanged
	}
	return core.Action{Path: p, Status: st, Existed: exists, Old: string(data), New: nw}, nil
}
