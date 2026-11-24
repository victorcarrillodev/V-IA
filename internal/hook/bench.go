package hook

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"
)

// BenchResult es la latencia de un tipo de decisión.
type BenchResult struct {
	Name          string
	P50, P95, Max time.Duration
}

// Bench mide, en un proyecto temporal, cuánto tarda Handle en cada tipo de acción.
// Es la sobrecarga que V-IA añade a cada llamada de herramienta del agente.
func Bench(n int) ([]BenchResult, error) {
	root, err := os.MkdirTemp("", "v-ia-bench")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(root)
	_ = exec.Command("git", "-C", root, "init", "-q").Run()
	_ = os.MkdirAll(filepath.Join(root, "odd", "tasks"), 0o755)
	_ = SaveState(root, State{Mode: ModeEdit})
	_ = SetTDD(root, true)
	mk := func(tool, file, cmd string) Input {
		in := Input{CWD: root, Event: "PreToolUse", Tool: tool}
		in.ToolInput.FilePath, in.ToolInput.Command = file, cmd
		return in
	}
	cases := []struct {
		name string
		in   Input
	}{
		{"prompt (clasificar petición)", Input{CWD: root, Event: "UserPromptSubmit", Prompt: "Corrige el texto del botón"}},
		{"Edit (permitido, test)", mk("Edit", filepath.Join(root, "a_test.go"), "")},
		{"Edit (TDD: consulta git)", mk("Edit", filepath.Join(root, "a.go"), "")},
		{"Bash (ls)", mk("Bash", "", "ls -la")},
		{"Bash (rm -rf /: bloqueado)", mk("Bash", "", "rm -rf /")},
	}
	var out []BenchResult
	for _, c := range cases {
		ds := make([]time.Duration, 0, n)
		for i := 0; i < n; i++ {
			t0 := time.Now()
			Handle(c.in)
			ds = append(ds, time.Since(t0))
		}
		sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
		out = append(out, BenchResult{c.name, ds[n/2].Round(time.Microsecond), ds[n*95/100].Round(time.Microsecond), ds[n-1].Round(time.Microsecond)})
	}
	return out, nil
}
