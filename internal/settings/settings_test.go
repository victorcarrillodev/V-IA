package settings

import (
	"strings"
	"testing"
)

const user = `{"model":"opus","hooks":{"Stop":[{"hooks":[{"type":"command","command":"echo hola"}]}]}}`

func TestMergeKeepsUserAndIsIdempotent(t *testing.T) {
	a, err := Merge(user, "/usr/bin/v-ia")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(a, `"model": "opus"`) || !strings.Contains(a, "echo hola") {
		t.Fatalf("se perdió configuración del usuario:\n%s", a)
	}
	if !Installed(a) {
		t.Fatal("debería estar instalado")
	}
	b, _ := Merge(a, "/usr/bin/v-ia")
	if a != b {
		t.Fatalf("no es idempotente:\n%s\n---\n%s", a, b)
	}
}

func TestMergeUpdatesExePath(t *testing.T) {
	a, _ := Merge("", "/old/v-ia")
	b, _ := Merge(a, "/new/v-ia")
	if strings.Contains(b, "/old/") || strings.Count(b, "/new/v-ia hook") != 3 {
		t.Fatalf("no actualizó la ruta:\n%s", b)
	}
}

func TestStripRestoresUser(t *testing.T) {
	a, _ := Merge(user, "/usr/bin/v-ia")
	s, err := Strip(a)
	if err != nil {
		t.Fatal(err)
	}
	if Installed(s) || strings.Contains(s, "v-ia") || !strings.Contains(s, "echo hola") {
		t.Fatalf("strip incorrecto:\n%s", s)
	}
}

func TestInvalidJSONIsRejected(t *testing.T) {
	if _, err := Merge("{no json", "v-ia"); err == nil {
		t.Fatal("debe rechazar JSON inválido")
	}
}
