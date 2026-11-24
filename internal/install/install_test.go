package install

import "testing"

func TestAddRemove(t *testing.T) {
	h := t.TempDir()
	if _, ok := Load(h); ok {
		t.Fatal("no debe haber registro")
	}
	if err := Add(h, Record{Version: "1", Preset: "standard", Agents: []string{"claude-code"}}); err != nil {
		t.Fatal(err)
	}
	if err := Add(h, Record{Version: "1", Preset: "standard", Agents: []string{"opencode"}, Components: []string{"context7"}}); err != nil {
		t.Fatal(err)
	}
	r, ok := Load(h)
	if !ok || len(r.Agents) != 2 || len(r.Components) != 1 {
		t.Fatalf("registro: %+v", r)
	}
	if err := Remove(h, []string{"opencode"}); err != nil {
		t.Fatal(err)
	}
	if r, _ = Load(h); len(r.Agents) != 1 || r.Agents[0] != "claude-code" {
		t.Fatalf("tras quitar opencode: %+v", r)
	}
	if err := Remove(h, []string{"claude-code"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := Load(h); ok {
		t.Fatal("sin agentes el registro se borra")
	}
	if err := Remove(h, []string{"x"}); err != nil {
		t.Fatal("quitar sin registro no es error")
	}
}
