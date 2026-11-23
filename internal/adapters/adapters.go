// Package adapters define cómo se renderiza ODD para cada agente.
package adapters

import (
	"os"
	"os/exec"
	"path/filepath"

	corefs "v-ia/core"
	"v-ia/internal/core"
)

const Version = "0.1.0"

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

func Find(name string) (Adapter, bool) {
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
