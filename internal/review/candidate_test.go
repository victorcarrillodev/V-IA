package review

import (
	"reflect"
	"testing"
)

// .gitignore con las carpetas de estado local (lo habitual) no debe impedir congelar el candidato.
func TestCandidateWithIgnoredStateDirs(t *testing.T) {
	root := setup(t, "true\n")
	write(t, root, ".gitignore", ".v-ia/\n.serena/\n")
	write(t, root, ".serena/cache", "x")
	write(t, root, "a.txt", "2")
	c, err := TakeCandidate(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{".gitignore", "a.txt"}; !reflect.DeepEqual(c.Paths, want) {
		t.Fatalf("rutas %v, quería %v", c.Paths, want)
	}
}

// Sin .gitignore, el estado local sigue fuera del candidato, tenga o no seguimiento en git.
func TestCandidateExcludesStateDirs(t *testing.T) {
	root := setup(t, "true\n")
	write(t, root, ".serena/tracked", "1")
	gitIn(t, root, "add", ".serena/tracked")
	gitIn(t, root, "commit", "-qm", "tracked")
	write(t, root, ".serena/tracked", "2")
	write(t, root, ".atl/registry", "x")
	write(t, root, "a.txt", "2")
	c, err := TakeCandidate(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a.txt"}; !reflect.DeepEqual(c.Paths, want) {
		t.Fatalf("rutas %v, quería %v", c.Paths, want)
	}
}
