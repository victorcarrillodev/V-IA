// Package install guarda qué instaló V-IA (agentes, preset, componentes) en
// ~/.v-ia/install.json, para que doctor y sync sepan qué debería haber.
package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Record struct {
	Version    string    `json:"version"`
	Preset     string    `json:"preset"`
	Agents     []string  `json:"agents"`
	Components []string  `json:"components,omitempty"`
	Hooks      bool      `json:"hooks"`
	Skills     bool      `json:"skills"`
	Updated    time.Time `json:"updated"`
}

func path(home string) string { return filepath.Join(home, ".v-ia", "install.json") }

// Load lee el registro; ok=false si no existe o está dañado.
func Load(home string) (Record, bool) {
	var r Record
	b, err := os.ReadFile(path(home))
	if err != nil || json.Unmarshal(b, &r) != nil {
		return Record{}, false
	}
	return r, true
}

func union(a, b []string) []string {
	m := map[string]bool{}
	for _, x := range append(append([]string{}, a...), b...) {
		m[x] = true
	}
	var out []string
	for x := range m {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

func save(home string, r Record) error {
	if err := os.MkdirAll(filepath.Dir(path(home)), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(r, "", "  ")
	return os.WriteFile(path(home), append(b, '\n'), 0o644)
}

// Record anota una instalación: fusiona agentes y componentes con lo ya registrado.
func Add(home string, n Record) error {
	old, _ := Load(home)
	n.Agents = union(old.Agents, n.Agents)
	n.Components = union(old.Components, n.Components)
	n.Hooks = n.Hooks || old.Hooks // lo más completo que se haya instalado en cualquiera de los agentes
	n.Skills = n.Skills || old.Skills
	return save(home, n)
}

// Remove retira agentes del registro; si no queda ninguno, borra el archivo.
func Remove(home string, agents []string) error {
	old, ok := Load(home)
	if !ok {
		return nil
	}
	drop := map[string]bool{}
	for _, a := range agents {
		drop[a] = true
	}
	var keep []string
	for _, a := range old.Agents {
		if !drop[a] {
			keep = append(keep, a)
		}
	}
	if len(keep) == 0 {
		err := os.Remove(path(home))
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	old.Agents = keep
	return save(home, old)
}
