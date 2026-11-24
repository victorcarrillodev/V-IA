package telemetry

import "testing"

func TestOffByDefaultAndCounts(t *testing.T) {
	h := t.TempDir()
	Record(h, "block:read-edit")
	if len(Load(h).Counts) != 0 {
		t.Fatal("apagado por defecto: no debe registrar nada")
	}
	if err := Set(h, true); err != nil {
		t.Fatal(err)
	}
	Record(h, "block:read-edit")
	Record(h, "block:read-edit")
	Record(h, "block:delivery")
	d := Load(h)
	if d.Counts["block:read-edit"] != 2 || d.Counts["block:delivery"] != 1 || len(d.Sorted()) != 2 {
		t.Fatalf("%v", d.Counts)
	}
	if err := Reset(h); err != nil || len(Load(h).Counts) != 0 {
		t.Fatal("reset")
	}
	_ = Set(h, false)
	Record(h, "x")
	if len(Load(h).Counts) != 0 {
		t.Fatal("tras apagar no registra")
	}
}
