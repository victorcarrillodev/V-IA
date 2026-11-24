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

func TestPlanContext7SinNode(t *testing.T) {
	fakePath(t, "claude") // sin npx ni npm: Context7 usa el MCP remoto
	steps, _, err := Plan(t.TempDir(), []string{Context7})
	if err != nil || len(steps) != 1 {
		t.Fatalf("steps=%v err=%v", steps, err)
	}
	want := "claude mcp add --scope user --transport http context7 " + Context7URL
	if steps[0].String() != want {
		t.Fatalf("comando=%q", steps[0])
	}
}

func TestPlanCodeGraphInstalaConNpmSiFalta(t *testing.T) {
	fakePath(t, "claude", "npm")
	steps, notes, err := Plan(t.TempDir(), []string{CodeGraph})
	if err != nil || len(steps) != 2 {
		t.Fatalf("steps=%v err=%v", steps, err)
	}
	if steps[0].Name != "npm" || !strings.Contains(steps[0].String(), CodeGraphPkg) || steps[1].String() != "codegraph install --yes --target=claude" {
		t.Fatalf("pasos inesperados: %v", steps)
	}
	if !strings.Contains(strings.Join(notes, " "), "CLAUDE.md") {
		t.Fatal("debe avisar de que escribe en CLAUDE.md")
	}
}

func TestPlanCodeGraphSinNpmUsaSuInstalador(t *testing.T) {
	fakePath(t, "claude", "curl")
	steps, _, err := Plan(t.TempDir(), []string{CodeGraph})
	if err != nil || len(steps) != 2 || steps[0].Name != "sh" || !strings.Contains(steps[0].String(), CodeGraphScript) {
		t.Fatalf("steps=%v err=%v", steps, err)
	}
}

func TestPlanCodeGraphYaPresenteOmiteLaInstalacion(t *testing.T) {
	fakePath(t, "claude", "codegraph")
	steps, _, err := Plan(t.TempDir(), []string{CodeGraph})
	if err != nil || len(steps) != 1 || steps[0].Name != "codegraph" {
		t.Fatalf("steps=%v err=%v", steps, err)
	}
}

func TestPlanEngramSeInstalaSiFalta(t *testing.T) {
	fakePath(t, "go")
	steps, _, err := Plan(t.TempDir(), []string{Engram})
	if err != nil || len(steps) != 2 {
		t.Fatalf("steps=%v err=%v", steps, err)
	}
	if steps[0].Name != "go" || !strings.Contains(steps[0].String(), EngramPkg) || len(steps[0].Env) != 1 || !strings.HasPrefix(steps[0].Env[0], "GOBIN=") {
		t.Fatalf("go install mal: %+v", steps[0])
	}
	if steps[1].String() != "engram setup claude-code" {
		t.Fatalf("setup: %v", steps[1])
	}
	fakePath(t, "brew")
	if steps, _, _ = Plan(t.TempDir(), []string{Engram}); steps[0].Name != "brew" {
		t.Fatalf("con brew debía usar brew: %v", steps)
	}
	fakePath(t, "engram")
	if steps, _, _ = Plan(t.TempDir(), []string{Engram}); len(steps) != 1 {
		t.Fatalf("con engram presente solo se configura: %v", steps)
	}
}

func TestPlanSoloFallaLoQueNoSePuedeInstalar(t *testing.T) {
	fakePath(t) // nada en PATH
	_, _, err := Plan(t.TempDir(), []string{Context7, CodeGraph, Engram})
	if !IsMissing(err) {
		t.Fatalf("debía faltar algo: %v", err)
	}
	for _, w := range []string{"claude", "engram", "go"} {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("el error no menciona %s: %v", w, err)
		}
	}
	// sin claude no se puede registrar nada, pero Node ya no es requisito
	fakePath(t, "go", "curl")
	if _, _, err = Plan(t.TempDir(), []string{Engram}); err != nil {
		t.Fatalf("engram con go no debía fallar: %v", err)
	}
}

func TestPlanMissingDevuelveTambienLosPasos(t *testing.T) {
	fakePath(t) // sin claude
	steps, _, err := Plan(t.TempDir(), []string{Context7})
	if !IsMissing(err) || len(steps) != 1 {
		t.Fatalf("steps=%v err=%v", steps, err)
	}
}

func TestPlanIdempotente(t *testing.T) {
	fakePath(t, "claude")
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"context7":{}}}`), 0o644)
	steps, notes, err := Plan(home, []string{Context7})
	if err != nil || len(steps) != 0 || len(notes) != 1 {
		t.Fatalf("steps=%v notes=%v err=%v", steps, notes, err)
	}
}

func TestDeteccionNoSeEngañaConRutasOHistorial(t *testing.T) {
	fakePath(t, "claude")
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

func TestEngramYaConfiguradoSeDetecta(t *testing.T) {
	fakePath(t, "engram")
	home := t.TempDir()
	if Installed(home, Engram) {
		t.Fatal("sin registrar en mcpServers no está configurado")
	}
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"engram":{}}}`), 0o644)
	if !Installed(home, Engram) {
		t.Fatal("con binario y mcpServers.engram debía detectarse")
	}
	steps, notes, err := Plan(home, []string{Engram})
	if err != nil || len(steps) != 0 || len(notes) != 1 {
		t.Fatalf("debía ser idempotente: steps=%v notes=%v err=%v", steps, notes, err)
	}
}
