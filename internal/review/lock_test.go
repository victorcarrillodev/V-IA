package review

import (
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// Los revisores pueden registrar a la vez: ningún veredicto se puede perder.
func TestConcurrentAcksKeepEveryVerdict(t *testing.T) {
	root := highRepo(t)
	run(t, root, RunOptions{})
	var inputs []ReviewInput
	for _, l := range AllLenses {
		inputs = append(inputs, okInput(t, root, l))
	}
	var wg sync.WaitGroup
	for _, in := range inputs {
		wg.Add(1)
		go func(in ReviewInput) {
			defer wg.Done()
			if _, err := RecordReview(root, in); err != nil {
				t.Errorf("%s: %v", in.Lens, err)
			}
		}(in)
	}
	wg.Wait()
	r, err := Load(root)
	if err != nil || len(r.Reviews) != len(AllLenses) || r.State != StateApproved {
		t.Fatalf("esperaba %d veredictos y approved: %d %s %v", len(AllLenses), len(r.Reviews), r.State, err)
	}
}

// Con el recibo ocupado, ack y reset esperan y fallan sin tocarlo.
func TestBusyReceiptIsNotTouched(t *testing.T) {
	root := highRepo(t)
	run(t, root, RunOptions{})
	in := okInput(t, root, "risk")
	old := lockWait
	lockWait = 100 * time.Millisecond
	t.Cleanup(func() { lockWait = old })
	unlock, err := lock(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RecordReview(root, in); err == nil {
		t.Fatal("ack con el recibo ocupado debe fallar")
	}
	if Reset(root) == nil {
		t.Fatal("reset con el recibo ocupado debe fallar")
	}
	unlock()
	if r, err := Load(root); err != nil || len(r.Reviews) != 0 {
		t.Fatalf("el recibo debe seguir intacto: %v %+v", err, r.Reviews)
	}
	if _, err := RecordReview(root, in); err != nil {
		t.Fatalf("al soltar el bloqueo debe admitirlo: %v", err)
	}
}

// check y refute también respetan el bloqueo.
func TestCheckAndRefuteWaitForLock(t *testing.T) {
	root := highRepo(t)
	run(t, root, RunOptions{})
	old := lockWait
	lockWait = 100 * time.Millisecond
	t.Cleanup(func() { lockWait = old })
	unlock, err := lock(root)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	_, cerr := Run(root, DetectCommands(root), RunOptions{})
	_, rerr := Refute(root, RefuteInput{})
	for op, err := range map[string]error{"check": cerr, "refute": rerr} {
		if err == nil || !strings.Contains(err.Error(), "en curso") {
			t.Errorf("%s con el recibo ocupado debe fallar por el bloqueo: %v", op, err)
		}
	}
}

// Un sistema sin soporte de bloqueo falla al momento y no culpa a otra operación.
func TestLockUnsupportedFailsFast(t *testing.T) {
	old := acquire
	acquire = func(*os.File) error { return errors.New("no soportado") }
	t.Cleanup(func() { acquire = old })
	start := time.Now()
	_, err := lock(t.TempDir())
	if err == nil || strings.Contains(err.Error(), "en curso") || time.Since(start) > 5*time.Second {
		t.Fatalf("debe fallar al momento: %v (%s)", err, time.Since(start))
	}
}

// En Windows un lector simultáneo hace fallar el rename durante un instante: se reintenta.
func TestSaveRetriesTransientRenameFailure(t *testing.T) {
	old, fails := rename, 3
	rename = func(a, b string) error {
		if fails > 0 {
			fails--
			return errors.New("en uso")
		}
		return old(a, b)
	}
	t.Cleanup(func() { rename = old })
	root := t.TempDir()
	if err := save(root, Receipt{Result: "pass"}); err != nil {
		t.Fatal(err)
	}
	if r, err := Load(root); err != nil || r.Result != "pass" {
		t.Fatalf("recibo no guardado: %v %+v", err, r)
	}
}
