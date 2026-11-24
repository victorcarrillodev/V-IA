package core

import (
	"os"
	"path/filepath"
)

// KeepBackups es cuántos backups sin fijar se conservan tras cada Apply.
const KeepBackups = 5

const pinFile = ".pinned"

// Pin marca un backup para que la poda no lo borre; Unpin lo revierte.
func Pin(dir string) error { return os.WriteFile(filepath.Join(dir, pinFile), nil, 0o600) }

func Unpin(dir string) error {
	if err := os.Remove(filepath.Join(dir, pinFile)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// IsPinned indica si el backup está fijado.
func IsPinned(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, pinFile))
	return err == nil
}

// Prune borra los backups sin fijar más antiguos, dejando los keep más recientes.
// Los fijados nunca se borran ni cuentan para el límite. Devuelve los borrados.
func Prune(backupRoot string, keep int) ([]string, error) {
	l, err := ListBackups(backupRoot)
	if err != nil {
		return nil, err
	}
	var unpinned []string
	for _, d := range l {
		if !IsPinned(d) {
			unpinned = append(unpinned, d)
		}
	}
	var removed []string
	for len(unpinned) > keep {
		if err := os.RemoveAll(unpinned[0]); err != nil {
			return removed, err
		}
		removed = append(removed, unpinned[0])
		unpinned = unpinned[1:]
	}
	return removed, nil
}
