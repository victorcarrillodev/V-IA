package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type captured struct {
	entry manifestEntry
	data  []byte
}

// Snapshot es el contenido de unos archivos antes de que algo externo los edite.
type Snapshot []captured

// Capture lee en memoria los archivos dados; los que no existen se anotan como tales.
func Capture(paths []string) (Snapshot, error) {
	var s Snapshot
	seen := map[string]bool{}
	for _, p := range paths {
		if seen[p] {
			continue
		}
		seen[p] = true
		c := captured{entry: manifestEntry{Path: p, Mode: 0o644}}
		b, err := os.ReadFile(p)
		switch {
		case err == nil:
			st, err := os.Stat(p)
			if err != nil {
				return nil, err
			}
			c.entry.Existed, c.entry.Mode, c.data = true, uint32(st.Mode().Perm()), b
		case !errors.Is(err, os.ErrNotExist):
			return nil, err
		}
		s = append(s, c)
	}
	return s, nil
}

// Save lo guarda como un backup normal (restore lo entiende).
func (s Snapshot) Save(backupRoot string, now time.Time) (string, error) {
	dir := filepath.Join(backupRoot, now.UTC().Format("20060102T150405.000000000Z"))
	if err := os.MkdirAll(filepath.Join(dir, "files"), 0o700); err != nil {
		return "", err
	}
	m := manifest{Created: now.UTC().Format(time.RFC3339)}
	for i, c := range s {
		e := c.entry
		if e.Existed {
			e.File = fmt.Sprintf("files/%d", i)
			if err := os.WriteFile(filepath.Join(dir, e.File), c.data, 0o600); err != nil {
				return "", err
			}
		}
		m.Entries = append(m.Entries, e)
	}
	mj, _ := json.MarshalIndent(m, "", "  ")
	return dir, os.WriteFile(filepath.Join(dir, "manifest.json"), mj, 0o600)
}
