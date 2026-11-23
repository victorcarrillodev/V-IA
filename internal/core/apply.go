package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type manifestEntry struct {
	Path    string `json:"path"`
	Existed bool   `json:"existed"`
	Mode    uint32 `json:"mode"`
	File    string `json:"file,omitempty"` // copia dentro del backup
}

type manifest struct {
	Created string          `json:"created"`
	Entries []manifestEntry `json:"entries"`
}

// writeHook permite simular fallos en tests.
var writeHook = writeAtomic

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".v-ia-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Apply escribe el plan: backup previo, escritura atómica y rollback si algo falla.
// Devuelve el directorio de backup ("" si no había cambios).
func Apply(p Plan, backupRoot string, now time.Time) (string, error) {
	if len(p.Conflicts()) > 0 {
		return "", fmt.Errorf("%d archivo(s) con bloque editado a mano; usa force para sobrescribir", len(p.Conflicts()))
	}
	changes := p.Changes()
	if len(changes) == 0 {
		return "", nil
	}
	dir := filepath.Join(backupRoot, now.UTC().Format("20060102T150405.000000000Z"))
	if err := os.MkdirAll(filepath.Join(dir, "files"), 0o755); err != nil {
		return "", err
	}
	m := manifest{Created: now.UTC().Format(time.RFC3339)}
	for i, a := range changes {
		e := manifestEntry{Path: a.Path, Existed: a.Existed, Mode: 0o644}
		if a.Existed {
			st, err := os.Stat(a.Path)
			if err != nil {
				return "", err
			}
			e.Mode = uint32(st.Mode().Perm())
			e.File = fmt.Sprintf("files/%d", i)
			if err := os.WriteFile(filepath.Join(dir, e.File), []byte(a.Old), 0o600); err != nil {
				return "", err
			}
		}
		m.Entries = append(m.Entries, e)
	}
	mj, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), mj, 0o600); err != nil {
		return "", err
	}
	for i, a := range changes {
		if err := writeHook(a.Path, []byte(a.New), os.FileMode(m.Entries[i].Mode)); err != nil {
			rbErr := restoreManifest(dir, m)
			return dir, errors.Join(fmt.Errorf("escribiendo %s: %w (rollback aplicado)", a.Path, err), rbErr)
		}
	}
	return dir, nil
}

func restoreManifest(dir string, m manifest) error {
	var errs []error
	for _, e := range m.Entries {
		if !e.Existed {
			if err := os.Remove(e.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
				errs = append(errs, err)
			}
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.File))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := writeAtomic(e.Path, data, os.FileMode(e.Mode)); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Restore devuelve los archivos al estado guardado en el backup.
func Restore(backupDir string) error {
	raw, err := os.ReadFile(filepath.Join(backupDir, "manifest.json"))
	if err != nil {
		return err
	}
	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	return restoreManifest(backupDir, m)
}

// ListBackups devuelve los backups ordenados del más antiguo al más reciente.
func ListBackups(backupRoot string) ([]string, error) {
	ents, err := os.ReadDir(backupRoot)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			out = append(out, filepath.Join(backupRoot, e.Name()))
		}
	}
	sort.Strings(out)
	return out, nil
}

// Latest devuelve el backup más reciente.
func Latest(backupRoot string) (string, error) {
	l, err := ListBackups(backupRoot)
	if err != nil {
		return "", err
	}
	if len(l) == 0 {
		return "", errors.New("no hay backups")
	}
	return l[len(l)-1], nil
}
