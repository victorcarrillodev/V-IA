package review

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Mode resuelve el interruptor RDD: ON por defecto; un OFF explícito gana.
// El OFF de un clon (proyecto) no anula el OFF global y viceversa: cualquiera apaga.
func Mode(root, home string) (on bool, source string) {
	if g := globalMode(home); g == "off" {
		return false, "global"
	}
	if b, err := os.ReadFile(filepath.Join(root, ".v-ia", "review-mode")); err == nil && strings.TrimSpace(string(b)) == "off" {
		return false, "project"
	}
	return true, "default"
}

type config struct {
	Review string `json:"review,omitempty"`
}

func cfgPath(home string) string { return filepath.Join(home, ".v-ia", "config.json") }

func globalMode(home string) string {
	var c config
	if b, err := os.ReadFile(cfgPath(home)); err == nil {
		_ = json.Unmarshal(b, &c)
	}
	return c.Review
}

// SetMode guarda "on" u "off" en el ámbito global o de proyecto.
func SetMode(root, home, scope string, on bool) error {
	v := "on"
	if !on {
		v = "off"
	}
	switch scope {
	case "global":
		b, _ := json.Marshal(config{Review: v})
		if err := os.MkdirAll(filepath.Dir(cfgPath(home)), 0o755); err != nil {
			return err
		}
		return os.WriteFile(cfgPath(home), b, 0o644)
	case "project":
		p := filepath.Join(root, ".v-ia", "review-mode")
		if on {
			if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		return os.WriteFile(p, []byte("off\n"), 0o644)
	}
	return errors.New("ámbito inválido: usa global o project")
}
