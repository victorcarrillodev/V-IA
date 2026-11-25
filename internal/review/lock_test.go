package review

import (
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
