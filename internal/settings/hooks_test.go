package settings

import (
	"strings"
	"testing"
)

func TestHooksMergeStripRoundTripAllAgents(t *testing.T) {
	for name, sp := range Specs {
		old := `{"theme":"dark","hooks":{"` + sp.Events[0].Name + `":[{"command":"/otro/hook.sh","hooks":[{"type":"command","command":"/otro/hook.sh"}]}]}}`
		nw, err := HooksMerge(old, "/bin/v-ia", sp)
		if err != nil {
			t.Fatal(name, err)
		}
		if !HooksInstalled(nw) || !strings.Contains(nw, `"theme": "dark"`) || !strings.Contains(nw, "/otro/hook.sh") || !strings.Contains(nw, "/bin/v-ia hook") {
			t.Fatalf("%s: merge perdió algo: %s", name, nw)
		}
		if again, _ := HooksMerge(nw, "/bin/v-ia", sp); again != nw {
			t.Fatalf("%s: no es idempotente", name)
		}
		back, err := HooksStrip(nw)
		if err != nil || HooksInstalled(back) || !strings.Contains(back, "/otro/hook.sh") || !strings.Contains(back, `"theme"`) {
			t.Fatalf("%s strip: %v %s", name, err, back)
		}
		if _, err := HooksMerge("{ roto", "/x", sp); err == nil {
			t.Fatal(name, "JSON inválido debe rechazarse")
		}
	}
}

func TestCursorShapeIsFlatAndVersioned(t *testing.T) {
	nw, _ := HooksMerge("", "/bin/v-ia", Specs["cursor"])
	if !strings.Contains(nw, `"version": 1`) || strings.Contains(nw, `"type": "command"`) {
		t.Fatalf("Cursor usa forma plana con version 1: %s", nw)
	}
}

func TestHookCommandQuotesPathsWithSpaces(t *testing.T) {
	nw, err := HooksMerge("", "/home/u/mi carpeta/v-ia", Specs["codex"])
	if err != nil || !strings.Contains(nw, `\"/home/u/mi carpeta/v-ia\" hook`) {
		t.Fatalf("la ruta con espacios debe ir entrecomillada: %v %s", err, nw)
	}
	if !HooksInstalled(nw) {
		t.Fatal("debe reconocer su propia entrada entrecomillada")
	}
	if back, _ := HooksStrip(nw); HooksInstalled(back) {
		t.Fatal("strip debe retirarla")
	}
}
