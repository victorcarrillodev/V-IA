package extras

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakePath(t *testing.T, bins ...string) {
	t.Helper()
	old := LookPath
	LookPath = func(b string) (string, error) {
		for _, x := range bins {
			if x == b {
				return "/fake/" + b, nil
			}
		}
		return "", errors.New("no está")
	}
	t.Cleanup(func() { LookPath = old })
}

func TestPlanContext7(t *testing.T) {
	fakePath(t, "claude", "npx")
	steps, _, err := Plan(t.TempDir(), []string{Context7})
	if err != nil || len(steps) != 1 {
		t.Fatalf("steps=%v err=%v", steps, err)
	}
	want := "claude mcp add --scope user context7 -- npx -y " + Context7Pkg
	if steps[0].String() != want {
		t.Fatalf("comando=%q", steps[0])
	}
}

func TestPlanCodeGraphInstalaSiFalta(t *testing.T) {
	fakePath(t, "claude", "npm")
	steps, notes, err := Plan(t.TempDir(), []string{CodeGraph})
	if err != nil || len(steps) != 2 {
		t.Fatalf("steps=%v err=%v", steps, err)
	}
	if steps[0].Name != "npm" || steps[1].String() != "codegraph install --yes --target=claude" {
		t.Fatalf("pasos inesperados: %v", steps)
	}
	if !strings.Contains(strings.Join(notes, " "), "CLAUDE.md") {
		t.Fatal("debe avisar de que escribe en CLAUDE.md")
	}
}

func TestPlanCodeGraphYaPresenteOmiteNpm(t *testing.T) {
	fakePath(t, "claude", "codegraph")
	steps, _, err := Plan(t.TempDir(), []string{CodeGraph})
	if err != nil || len(steps) != 1 || steps[0].Name != "codegraph" {
		t.Fatalf("steps=%v err=%v", steps, err)
	}
}

func TestPlanFaltanRequisitos(t *testing.T) {
	fakePath(t) // nada en PATH
	_, _, err := Plan(t.TempDir(), []string{Context7, CodeGraph, Engram})
	if !IsMissing(err) {
		t.Fatal("debía fallar")
	}
	for _, w := range []string{"claude", "npx", "npm", "engram"} {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("el error no menciona %s: %v", w, err)
		}
	}
}

func TestPlanIdempotente(t *testing.T) {
	fakePath(t, "claude", "npx")
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"context7":{}}}`), 0o644)
	steps, notes, err := Plan(home, []string{Context7})
	if err != nil || len(steps) != 0 || len(notes) != 1 {
		t.Fatalf("steps=%v notes=%v err=%v", steps, notes, err)
	}
}

func TestPlanMissingDevuelveTambienLosPasos(t *testing.T) {
	fakePath(t, "claude") // sin npx
	steps, _, err := Plan(t.TempDir(), []string{Context7})
	if !IsMissing(err) || len(steps) != 1 {
		t.Fatalf("steps=%v err=%v", steps, err)
	}
}

func TestDeteccionNoSeEngañaConRutasOHistorial(t *testing.T) {
	fakePath(t, "claude", "npx")
	home := t.TempDir()
	// "context7" solo aparece como ruta de proyecto y en otro servidor: NO está registrado
	os.WriteFile(filepath.Join(home, ".claude.json"),
		[]byte(`{"projects":{"/home/x/context7":{"mcpServers":{}}},"mcpServers":{"otro":{"args":["context7"]}}}`), 0o644)
	if Installed(home, Context7) {
		t.Fatal("falso positivo: context7 no está en mcpServers")
	}
	steps, _, err := Plan(home, []string{Context7})
	if err != nil || len(steps) != 1 {
		t.Fatalf("debía planificar el registro: steps=%v err=%v", steps, err)
	}
}

func TestPlanDesconocido(t *testing.T) {
	if _, _, err := Plan(t.TempDir(), []string{"nope"}); err == nil {
		t.Fatal("debía rechazar un id desconocido")
	}
}

func TestRunSeDetieneEnElPrimerError(t *testing.T) {
	var ran []string
	old := Exec
	Exec = func(s Step, _ io.Writer) error {
		ran = append(ran, s.Name)
		if s.Name == "a" {
			return errors.New("falló")
		}
		return nil
	}
	t.Cleanup(func() { Exec = old })
	err := Run([]Step{{Name: "a"}, {Name: "b"}}, io.Discard)
	if err == nil || len(ran) != 1 {
		t.Fatalf("ran=%v err=%v", ran, err)
	}
}
