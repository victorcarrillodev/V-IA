package review

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// lockWait es cuánto espera una operación a que termine otra sobre el mismo recibo.
var lockWait = 2 * time.Minute

var acquire = tryLock // sustituible en pruebas

// lock serializa las operaciones que leen y reescriben el recibo (check, ack, refute, reset). El
// bloqueo es del sistema operativo: se libera solo aunque el proceso muera.
func lock(root string) (func(), error) {
	p := filepath.Join(root, ".v-ia", "receipt.lock")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	for deadline := time.Now().Add(lockWait); ; time.Sleep(50 * time.Millisecond) {
		err := acquire(f)
		if err == nil {
			return func() { f.Close() }, nil // cerrar el archivo suelta el bloqueo
		}
		if !isBusy(err) {
			f.Close()
			return nil, fmt.Errorf("no puedo bloquear %s: %w", p, err)
		}
		if time.Now().After(deadline) {
			f.Close()
			return nil, errors.New("otra operación de revisión sigue en curso en este proyecto: reintenta cuando termine (" + err.Error() + ")")
		}
	}
}
