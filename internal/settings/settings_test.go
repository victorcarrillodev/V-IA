package settings

import (
	"strings"
	"testing"
)

const user = `{"model":"opus","hooks":{"Stop":[{"hooks":[{"type":"command","command":"echo hola"}]}]}}`

func TestMergeKeepsUserAndIsIdempotent(t *testing.T) {
	a, err := Merge(user, "/usr/bin/v-ia", true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(a, `"model": "opus"`) || !strings.Contains(a, "echo hola") {
		t.Fatalf("se perdió configuración del usuario:\n%s", a)
	}
	if !Installed(a) {
		t.Fatal("debería estar instalado")
	}
	b, _ := Merge(a, "/usr/bin/v-ia", true)
	if a != b {
		t.Fatalf("no es idempotente:\n%s\n---\n%s", a, b)
	}
}

func TestMergeUpdatesExePath(t *testing.T) {
	a, _ := Merge("", "/old/v-ia", true)
	b, _ := Merge(a, "/new/v-ia", true)
	if strings.Contains(b, "/old/") || strings.Count(b, "/new/v-ia hook") != 4 {
		t.Fatalf("no actualizó la ruta:\n%s", b)
	}
}

func TestStripRestoresUser(t *testing.T) {
	a, _ := Merge(user, "/usr/bin/v-ia", true)
	s, err := Strip(a)
	if err != nil {
		t.Fatal(err)
	}
	if Installed(s) || strings.Contains(s, "v-ia") || !strings.Contains(s, "echo hola") {
		t.Fatalf("strip incorrecto:\n%s", s)
	}
}

func TestInvalidJSONIsRejected(t *testing.T) {
	if _, err := Merge("{no json", "v-ia", true); err == nil {
		t.Fatal("debe rechazar JSON inválido")
	}
}

func TestDenySeededOnlyWhenAbsent(t *testing.T) {
	a, _ := Merge("", "v-ia", true)
	if !strings.Contains(a, "Read(~/.ssh/**)") {
		t.Fatalf("falta la deny-list:\n%s", a)
	}
	own := `{"permissions":{"deny":[]}}`
	b, _ := Merge(own, "v-ia", true)
	if strings.Contains(b, ".ssh") {
		t.Fatalf("no debe pisar una lista vacía del usuario:\n%s", b)
	}
	c, _ := Merge("", "v-ia", false)
	if strings.Contains(c, "permissions") {
		t.Fatal("sin perms no debe tocar permissions")
	}
}
