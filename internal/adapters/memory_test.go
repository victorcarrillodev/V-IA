package adapters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Con Engram configurado, su protocolo ya está cargado: V-IA no lo repite.
func TestMemoryProtocolNotDuplicatedWithEngram(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("CODEX_HOME", "")
	home := t.TempDir()
	a, _ := Find("codex")
	full, err := a.Block(home, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(full.Content, "Qué guardar sin que te lo pidan") {
		t.Fatal("sin Engram va el protocolo completo")
	}
	// para un agente distinto de Codex basta con que Engram esté planeado
	oc, _ := Find("opencode")
	planned, _ := oc.Block(t.TempDir(), true)
	ocFull, _ := oc.Block(t.TempDir(), false)
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".codex", "config.toml"), []byte("[mcp_servers.engram]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	present, _ := a.Block(home, false)
	for name, b := range map[string]string{"planeado": planned.Content, "configurado": present.Content} {
		if strings.Contains(b, "Qué guardar sin que te lo pidan") {
			t.Errorf("%s: protocolo duplicado", name)
		}
		// lo propio de V-IA, que Engram no sabe, se mantiene
		for _, want := range []string{"odd/<feature>/tasks", "gana el código"} {
			if !strings.Contains(b, want) {
				t.Errorf("%s: falta %q", name, want)
			}
		}
		base := full.Content
		if name == "planeado" {
			base = ocFull.Content
		}
		if len(base)-len(b) < 1000 {
			t.Errorf("%s: ahorra solo %d bytes", name, len(full.Content)-len(b))
		}
	}
}
