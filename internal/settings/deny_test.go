package settings

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func denyOf(t *testing.T, existing string) []string {
	t.Helper()
	out, err := Merge(existing, "v-ia", false, true)
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		Permissions struct{ Deny []string }
	}
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatal(err)
	}
	return s.Permissions.Deny
}

func withDeny(l []string) string {
	b, _ := json.Marshal(map[string]any{"permissions": map[string]any{"deny": l}})
	return string(b)
}

// readOnly es la lista que sembraban las versiones anteriores.
func readOnly(l []string) (out []string) {
	for _, r := range l {
		if strings.HasPrefix(r, "Read(") {
			out = append(out, r)
		}
	}
	return out
}

func TestSensitivePathsDenyReadAndEdit(t *testing.T) {
	got := denyOf(t, "")
	has := map[string]bool{}
	for _, r := range got {
		has[r] = true
	}
	reads := readOnly(got)
	if len(reads) != 11 {
		t.Fatalf("la negación de lectura se mantiene: %d", len(reads))
	}
	for _, r := range reads {
		if e := "Edit(" + strings.TrimPrefix(r, "Read("); !has[e] {
			t.Errorf("falta %s", e)
		}
	}
}

// Una instalación anterior con la lista intacta recibe las reglas de edición.
func TestUntouchedOldDenyListIsUpgraded(t *testing.T) {
	want := denyOf(t, "")
	if got := denyOf(t, withDeny(readOnly(want))); !reflect.DeepEqual(got, want) {
		t.Fatalf("lista antigua sin actualizar: %v", got)
	}
}

// Lo que el usuario cambió no se toca.
func TestUserDenyListIsKept(t *testing.T) {
	old := readOnly(denyOf(t, ""))
	for _, mine := range [][]string{{}, old[1:], append(append([]string{}, old...), "Bash(curl:*)")} {
		if got := denyOf(t, withDeny(mine)); len(got) != len(mine) || len(mine) > 0 && !reflect.DeepEqual(got, mine) {
			t.Fatalf("se tocó la lista del usuario: %v -> %v", mine, got)
		}
	}
}
