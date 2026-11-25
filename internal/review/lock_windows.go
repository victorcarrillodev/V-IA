package review

import (
	"os"

	"golang.org/x/sys/windows"
)

// El rango bloqueado está lejos del contenido: en Windows el bloqueo impide también leer esos bytes.
func tryLock(f *os.File) error {
	return windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, &windows.Overlapped{Offset: 1 << 30})
}
