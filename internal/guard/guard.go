// Package guard detecta cambios en el árbol de trabajo durante una petición de solo lectura.
package guard

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

var skip = map[string]bool{".git": true, ".v-ia": true, "node_modules": true, ".serena": true}

// Snapshot mapea ruta relativa -> sha256 del contenido.
type Snapshot map[string]string

func Take(root string) (Snapshot, error) {
	s := Snapshot{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skip[d.Name()] && p != root {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		h := sha256.Sum256(b)
		rel, _ := filepath.Rel(root, p)
		s[rel] = hex.EncodeToString(h[:])
		return nil
	})
	return s, err
}

func path(root string) string { return filepath.Join(root, ".v-ia", "guard.json") }

// Start guarda el estado actual como referencia.
func Start(root string) error {
	s, err := Take(root)
	if err != nil {
		return err
	}
	b, _ := json.Marshal(s)
	if err := os.MkdirAll(filepath.Dir(path(root)), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path(root), b, 0o644)
}

// Diff compara el árbol actual con la referencia y devuelve las rutas cambiadas.
func Diff(root string) ([]string, error) {
	raw, err := os.ReadFile(path(root))
	if err != nil {
		return nil, err
	}
	var before Snapshot
	if err := json.Unmarshal(raw, &before); err != nil {
		return nil, err
	}
	after, err := Take(root)
	if err != nil {
		return nil, err
	}
	var ch []string
	for p, h := range after {
		if bh, ok := before[p]; !ok {
			ch = append(ch, "+ "+p)
		} else if bh != h {
			ch = append(ch, "~ "+p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			ch = append(ch, "- "+p)
		}
	}
	sort.Strings(ch)
	return ch, nil
}
