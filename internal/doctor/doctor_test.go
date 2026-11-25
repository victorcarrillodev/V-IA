package doctor

import (
	"strings"
	"testing"
)

func messages(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("V_IA_HOME", home)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("CODEX_HOME", "")
	var b strings.Builder
	for _, r := range Run(home) {
		b.WriteString(r.Level.Symbol() + " " + r.Msg + "\n")
	}
	return b.String()
}

// Los cuatro agentes principales se informan aunque no se detecten.
func TestReportsPrimaryAgents(t *testing.T) {
	out := messages(t)
	for _, n := range []string{"claude-code", "opencode", "codex", "antigravity"} {
		if !strings.Contains(out, "! "+n+": no encontrado") {
			t.Errorf("falta %s:\n%s", n, out)
		}
	}
}

// Solo Claude Code y OpenCode tienen el bloqueo comprobado con el agente real.
func TestDoesNotClaimUnverifiedEnforcement(t *testing.T) {
	out := messages(t)
	for _, n := range []string{"codex"} {
		if !strings.Contains(out, "! "+n+": hooks sin verificar con el agente real") {
			t.Errorf("%s debe declararse sin verificar:\n%s", n, out)
		}
		if strings.Contains(out, n+": ODD se hace cumplir") {
			t.Errorf("%s no debe prometer bloqueo", n)
		}
	}
	if !strings.Contains(out, "claude-code: ODD se hace cumplir") {
		t.Errorf("claude-code mantiene su mensaje:\n%s", out)
	}
	if !strings.Contains(out, "✓ opencode: ODD se hace cumplir") {
		t.Errorf("opencode está verificado:\n%s", out)
	}
	if !strings.Contains(out, "! antigravity: solo instrucciones") {
		t.Errorf("antigravity no tiene hooks:\n%s", out)
	}
}
