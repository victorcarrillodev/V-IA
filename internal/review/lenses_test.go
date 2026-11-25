package review

import (
	"reflect"
	"testing"
)

func run(t *testing.T, root string, o RunOptions) Receipt {
	t.Helper()
	r, err := Run(root, DetectCommands(root), o)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

var one = RunOptions{Lenses: []string{"risk"}, LensReason: "solo auth"}

func TestRepeatKeepsChosenLenses(t *testing.T) {
	root := highRepo(t)
	run(t, root, one)
	for _, code := range []string{"package auth\n", "package auth // v2\n"} {
		write(t, root, "internal/auth/login.go", code)
		r := run(t, root, RunOptions{})
		if !reflect.DeepEqual(r.SelectedLenses, []string{"risk"}) || r.LensReason != "solo auth" || r.State != StateChecking {
			t.Fatalf("la selección debe conservarse: %v %q %s", r.SelectedLenses, r.LensReason, r.State)
		}
	}
}

func TestOpenReviewLensesCanOnlyGrow(t *testing.T) {
	root := highRepo(t)
	run(t, root, RunOptions{})
	if _, err := Run(root, DetectCommands(root), one); err == nil {
		t.Fatal("reducir lentes en una revisión abierta debe fallar")
	}
	if r, _ := Load(root); len(r.SelectedLenses) != 4 {
		t.Fatalf("el recibo no debe cambiar: %v", r.SelectedLenses)
	}
	root = highRepo(t)
	run(t, root, one)
	r := run(t, root, RunOptions{Lenses: []string{"risk", "reliability"}, LensReason: "y fiabilidad"})
	if !reflect.DeepEqual(r.SelectedLenses, []string{"risk", "reliability"}) {
		t.Fatalf("añadir sí se permite: %v", r.SelectedLenses)
	}
}

func TestRepeatKeepsEscalation(t *testing.T) {
	root := setup(t, "true\n")
	write(t, root, "a.txt", "2")
	run(t, root, RunOptions{EscItem: 3, EscReason: "formato"})
	r := run(t, root, RunOptions{})
	if r.Risk.Tier != TierHigh || r.State != StateChecking || len(r.SelectedLenses) != 4 {
		t.Fatalf("repetir sin flags no puede bajar el riesgo: %s %s %v", r.Risk.Tier, r.State, r.SelectedLenses)
	}
}

// Una revisión cerrada (aprobada) no condiciona el cambio siguiente.
func TestNewReviewStartsFresh(t *testing.T) {
	root := highRepo(t)
	run(t, root, one)
	if _, err := RecordReview(root, okInput(t, root, "risk")); err != nil {
		t.Fatal(err)
	}
	write(t, root, "internal/auth/login.go", "package auth // otro\n")
	if r := run(t, root, RunOptions{}); len(r.SelectedLenses) != 4 || r.LensReason != "" {
		t.Fatalf("linaje nuevo con lentes por defecto: %v %q", r.SelectedLenses, r.LensReason)
	}
}

// Repetir el mismo candidato ya aprobado conserva la selección con la que se aprobó.
func TestSameApprovedCandidateKeepsSelection(t *testing.T) {
	root := highRepo(t)
	run(t, root, one)
	if _, err := RecordReview(root, okInput(t, root, "risk")); err != nil {
		t.Fatal(err)
	}
	if r := run(t, root, RunOptions{}); !reflect.DeepEqual(r.SelectedLenses, []string{"risk"}) {
		t.Fatalf("mismo candidato, mismas lentes: %v", r.SelectedLenses)
	}
}

func TestKeepLensesWhenRiskChanges(t *testing.T) {
	prev := Receipt{SelectedLenses: []string{"risk"}, LensReason: "solo auth",
		Risk: Assessment{Tier: TierMedium, ReviewDue: true, ReviewDueReason: "high_risk"}}

	up := Receipt{SelectedLenses: DefaultLenses(TierHigh), Risk: Assessment{Tier: TierHigh, ReviewDue: true}}
	if err := keepLenses(&up, prev, false); err != nil || !reflect.DeepEqual(up.SelectedLenses, AllLenses) {
		t.Fatalf("si el riesgo sube se añaden las lentes por defecto: %v %v", up.SelectedLenses, err)
	}

	down := Receipt{Risk: Assessment{Tier: TierPassive}}
	if err := keepLenses(&down, prev, false); err != nil || !down.Risk.ReviewDue || down.Risk.ReviewDueReason != "high_risk" || initialState(Receipt{Result: "pass", Risk: down.Risk, SelectedLenses: down.SelectedLenses}) != StateChecking {
		t.Fatalf("la revisión debida no desaparece al bajar el riesgo: %+v %v", down, err)
	}
}
