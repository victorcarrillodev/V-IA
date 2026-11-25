package adapters

import (
	"strings"
	"testing"
)

func block(t *testing.T, name string) string {
	t.Helper()
	a, ok := Find(name)
	if !ok {
		t.Fatal(name)
	}
	b, err := a.Block(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	return b.Content
}

// Las reglas siempre cargadas dicen cuándo delegar, sin esperar a que se abra una skill.
func TestEveryAgentGetsDelegationRules(t *testing.T) {
	for _, n := range Primary {
		got := block(t, n)
		for _, want := range []string{"**Delegar.**", "~3 llamadas", "Exploración", "Escritores paralelos", "Respaldo de contexto", "Verificador independiente", "no tiene subagentes"} {
			if !strings.Contains(got, want) {
				t.Errorf("%s: falta %q", n, want)
			}
		}
		if len(got) > 9600 {
			t.Errorf("%s: %d bytes; las reglas se pagan en cada sesión", n, len(got))
		}
	}
}

func TestSubagentsListedOnlyWhereInstalled(t *testing.T) {
	for _, n := range Primary {
		a, _ := Find(n)
		if has := strings.Contains(block(t, n), "v-ia-explorer"); has != (a.AgentsDir != nil) {
			t.Errorf("%s: menciona subagentes=%v", n, has)
		}
	}
}

// Solo se promete bloqueo donde se comprobó con el agente real.
func TestBlockingPromisedOnlyWhereVerified(t *testing.T) {
	for _, n := range Primary {
		want := n == "claude-code" || n == "opencode"
		if has := strings.Contains(block(t, n), "BLOQUEAN"); has != want {
			t.Errorf("%s: promete bloqueo=%v", n, has)
		}
	}
}
