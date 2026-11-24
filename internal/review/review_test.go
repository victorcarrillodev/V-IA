package review

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitIn(t *testing.T, root string, args ...string) {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", root, "-c", "user.email=t@t", "-c", "user.name=t"}, args...)...)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

// setup crea un repositorio con un commit, un archivo y las comprobaciones dadas.
func setup(t *testing.T, cmds string) string {
	root := t.TempDir()
	gitIn(t, root, "init", "-q")
	os.WriteFile(filepath.Join(root, "a.txt"), []byte("1"), 0o644)
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-qm", "init")
	os.MkdirAll(filepath.Join(root, ".v-ia"), 0o755)
	os.WriteFile(filepath.Join(root, ".v-ia", "checks.txt"), []byte(cmds), 0o644)
	return root
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPassThenByteChangeInvalidates(t *testing.T) {
	root := setup(t, "true\n")
	r, err := Run(root, DetectCommands(root), RunOptions{})
	if err != nil || r.Result != "pass" || r.CandidateTree == "" {
		t.Fatalf("esperaba pass con candidato congelado: %v %+v", err, r)
	}
	if err := Verify(root, false); err != nil {
		t.Fatal(err)
	}
	write(t, root, "a.txt", "2")
	if Verify(root, false) == nil {
		t.Fatal("cambiar un byte debe invalidar el recibo")
	}
}

func TestIgnoredFilesDoNotInvalidate(t *testing.T) {
	root := setup(t, "true\n")
	write(t, root, ".gitignore", "build/\n")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-qm", "ignore")
	if _, err := Run(root, DetectCommands(root), RunOptions{}); err != nil {
		t.Fatal(err)
	}
	write(t, root, "build/out.bin", "artefacto")
	if err := Verify(root, false); err != nil {
		t.Fatalf("un archivo ignorado no debe invalidar el recibo: %v", err)
	}
}

func TestChangeDuringChecksIsStale(t *testing.T) {
	root := setup(t, "sh -c 'echo x > mutado.txt'\n")
	r, _ := Run(root, DetectCommands(root), RunOptions{})
	if r.Result != "stale" || Verify(root, false) == nil {
		t.Fatalf("un cambio durante las comprobaciones debe dar recibo stale: %+v", r.Result)
	}
}

func TestFailingCheckGivesFail(t *testing.T) {
	root := setup(t, "false\n")
	r, _ := Run(root, DetectCommands(root), RunOptions{})
	if r.Result != "fail" || Verify(root, false) == nil {
		t.Fatal("un check fallido no debe dar recibo válido")
	}
}

func TestNoChecksIsError(t *testing.T) {
	if _, err := Run(t.TempDir(), nil, RunOptions{}); err == nil {
		t.Fatal("sin comprobaciones no puede haber recibo")
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]int
		added map[string][]string
		tier  string
	}{
		{"docs", map[string]int{"README.md": 40}, nil, TierPassive},
		{"pequeño con test", map[string]int{"a.go": 10, "a_test.go": 10}, nil, TierPassive},
		{"sin tests y grande", map[string]int{"a.go": 60}, nil, TierMedium},
		{"muy amplio", map[string]int{"a.go": 200, "a_test.go": 5}, nil, TierMedium},
		{"ruta de auth", map[string]int{"internal/auth/login.go": 5, "internal/auth/login_test.go": 5}, nil, TierHigh},
		{"migración sql", map[string]int{"db/001.sql": 3}, nil, TierHigh},
		{"workflow", map[string]int{".github/workflows/ci.yml": 4}, nil, TierHigh},
		{"texto sensible en código", map[string]int{"a.go": 3}, map[string][]string{"a.go": {`password = "supersecreto123"`}}, TierHigh},
		{"dependencias", map[string]int{"go.mod": 2, "a.go": 3, "a_test.go": 3}, nil, TierMedium},
		{"sin cambios", map[string]int{}, nil, TierPassive},
		// regresiones del revisor: los tests y las docs no son evidencia de riesgo…
		{"test en carpeta auth no sube", map[string]int{"internal/auth/login_test.go": 30}, nil, TierPassive},
		{"texto sensible solo en un test", map[string]int{"a.go": 3, "a_test.go": 5}, map[string][]string{"a_test.go": {"exec.Command(\"ls\")"}}, TierPassive},
		// …pero un script bajo docs/ SÍ es código
		{"script bajo docs", map[string]int{"docs/deploy.sh": 3}, nil, TierHigh},
	}
	for _, c := range cases {
		if got := classify(c.files, c.added); got.Tier != c.tier {
			t.Errorf("%s: tier=%s, esperaba %s (%v)", c.name, got.Tier, c.tier, got.Reasons)
		}
	}
	if a := classify(map[string]int{"internal/auth/x.go": 1}, nil); !a.ReviewDue || a.ReviewDueReason != DueHighRisk {
		t.Fatalf("alto exige revisión: %+v", a)
	}
}

func TestAssessOutsideGitFailsClosed(t *testing.T) {
	a := Assess(t.TempDir(), "")
	if a.Tier != TierUnknown || !a.ReviewDue || a.ReviewDueReason != DueAssessmentFailed {
		t.Fatalf("una evaluación fallida nunca implica riesgo bajo: %+v", a)
	}
}

func TestUntrackedFilesCountRealLines(t *testing.T) {
	root := setup(t, "true\n")
	write(t, root, "nuevo.go", strings.Repeat("x := 1\n", 400))
	a := Assess(root, "")
	if a.Lines < 400 || a.Tier != TierMedium {
		t.Fatalf("un archivo nuevo de 400 líneas debe pesar 400: %+v", a)
	}
}

func TestEscalationAndLenses(t *testing.T) {
	a := classify(map[string]int{"a.go": 10, "a_test.go": 10}, nil)
	if a.Tier != TierPassive {
		t.Fatal("base pasiva")
	}
	if err := a.Escalate(7, "x"); err == nil {
		t.Fatal("ítem fuera de 1-6")
	}
	if err := a.Escalate(3, "cambia el formato de configuración"); err != nil || a.Tier != TierHigh || !a.ReviewDue {
		t.Fatalf("escalado: %+v %v", a, err)
	}
	if err := a.Escalate(2, "una\ndos"); err == nil {
		t.Fatal("la razón debe ser de una línea")
	}
	if l := DefaultLenses(TierHigh); len(l) != 4 {
		t.Fatal("high = 4 lentes")
	}
	if l := DefaultLenses(TierMedium); len(l) != 1 || l[0] != "reliability" {
		t.Fatal("medium = reliability")
	}
	if len(DefaultLenses(TierPassive)) != 0 {
		t.Fatal("passive = 0")
	}
	if _, err := SelectLenses(TierHigh, []string{"risk"}, ""); err == nil {
		t.Fatal("--lenses exige razón")
	}
	if _, err := SelectLenses(TierPassive, []string{"risk"}, "x"); err == nil {
		t.Fatal("passive no admite selección")
	}
	l, err := SelectLenses(TierHigh, []string{"reliability", "risk"}, "toca auth")
	if err != nil || len(l) != 2 || l[0] != "risk" || l[1] != "reliability" {
		t.Fatalf("orden canónico 4R: %v %v", l, err)
	}
}

// highRepo deja un cambio de riesgo alto con comprobaciones que pasan.
func highRepo(t *testing.T) string {
	root := setup(t, "true\n")
	write(t, root, "internal/auth/login.go", "package auth\n")
	return root
}

func okInput(t *testing.T, root, lens string, findings ...Finding) ReviewInput {
	r, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return ReviewInput{CandidateID: r.CandidateID, Lens: lens, Reviewer: "rev", Inspection: Inspection{Status: "complete", Paths: r.Paths},
		Findings: findings, Evidence: []string{"leí el diff completo"}}
}

func TestHighRiskNeedsEveryLens(t *testing.T) {
	root := highRepo(t)
	r, _ := Run(root, DetectCommands(root), RunOptions{})
	if r.Risk.Tier != TierHigh || len(r.SelectedLenses) != 4 || r.State != StateChecking {
		t.Fatalf("alto: %+v", r)
	}
	if Verify(root, true) == nil {
		t.Fatal("riesgo alto sin revisiones debe fallar")
	}
	if Verify(root, false) != nil {
		t.Fatal("con RDD apagado no exige revisión")
	}
	for i, l := range AllLenses {
		if _, err := RecordReview(root, okInput(t, root, l)); err != nil {
			t.Fatalf("lente %s: %v", l, err)
		}
		if i < 3 && Verify(root, true) == nil {
			t.Fatal("faltan lentes: no basta")
		}
	}
	if err := Verify(root, true); err != nil {
		t.Fatal(err)
	}
	if st := Status(root); st.State != PublicReady {
		t.Fatalf("aprobado => ready: %+v", st)
	}
	write(t, root, "a.txt", "cambio")
	if Verify(root, true) == nil || Status(root).State != PublicWorking {
		t.Fatal("cambiar el árbol invalida recibo y revisiones (working)")
	}
}

func TestReviewValidation(t *testing.T) {
	root := highRepo(t)
	Run(root, DetectCommands(root), RunOptions{})
	bad := okInput(t, root, "risk")
	bad.Inspection.Paths = nil
	if _, err := RecordReview(root, bad); err == nil || !strings.Contains(err.Error(), "manifiesto") {
		t.Fatalf("inspection debe cubrir el manifiesto: %v", err)
	}
	if _, err := RecordReview(root, okInput(t, root, "nada")); err == nil {
		t.Fatal("lente no seleccionada")
	}
	noProof := okInput(t, root, "risk", Finding{Location: "a:1", Severity: "BLOCKER", Claim: "x", CausalDisposition: "introduced"})
	if _, err := RecordReview(root, noProof); err == nil || !strings.Contains(err.Error(), "proof_refs") {
		t.Fatalf("un severo causal exige prueba: %v", err)
	}
	noCausal := okInput(t, root, "risk", Finding{Location: "a:1", Severity: "CRITICAL", Claim: "x"})
	if _, err := RecordReview(root, noCausal); err == nil {
		t.Fatal("un severo exige causal_disposition")
	}
	wrongID := okInput(t, root, "risk")
	wrongID.CandidateID = "otro"
	if _, err := RecordReview(root, wrongID); err == nil {
		t.Fatal("candidate_id debe coincidir")
	}
}

func TestVerdictIsComputedAndOneCorrectionThenEscalation(t *testing.T) {
	root := highRepo(t)
	opts := RunOptions{Lenses: []string{"risk"}, LensReason: "solo seguridad"}
	Run(root, DetectCommands(root), opts)
	// pre-existing no bloquea aunque sea BLOCKER
	pre := Finding{Location: "x:1", Severity: "BLOCKER", Claim: "ya estaba", CausalDisposition: "pre-existing"}
	r, err := RecordReview(root, okInput(t, root, "risk", pre))
	if err != nil || r.State != StateApproved || r.Reviews[0].Verdict != "approve" {
		t.Fatalf("un severo preexistente no bloquea: %+v %v", r, err)
	}
	// nueva ronda: bloqueante causal
	write(t, root, "internal/auth/login.go", "package auth // v2\n")
	Run(root, DetectCommands(root), opts)
	blocker := Finding{Location: "login.go:1", Severity: "BLOCKER", Claim: "salta la auth", CausalDisposition: "introduced", ProofRefs: []string{"go test -run X"}}
	r, err = RecordReview(root, okInput(t, root, "risk", blocker))
	if err != nil || r.State != StateCorrectionRequired || r.Reviews[0].Verdict != "changes" {
		t.Fatalf("bloqueante causal => correction_required: %+v %v", r, err)
	}
	if Verify(root, true) == nil || Status(root).State != PublicWorking {
		t.Fatal("en corrección no se puede entregar")
	}
	// la única corrección: se vuelve a comprobar (intento 2) y vuelve a fallar => escalado
	write(t, root, "internal/auth/login.go", "package auth // v3\n")
	r2, _ := Run(root, DetectCommands(root), opts)
	if r2.Attempt != 2 {
		t.Fatalf("tras correction_required el siguiente check es el intento 2: %d", r2.Attempt)
	}
	r, err = RecordReview(root, okInput(t, root, "risk", blocker))
	if err != nil || r.State != StateEscalated {
		t.Fatalf("segunda negativa => escalated: %+v %v", r, err)
	}
	if Status(root).State != PublicNeedsDecision {
		t.Fatal("escalado = needs_decision")
	}
	if _, err := Run(root, DetectCommands(root), opts); err == nil {
		t.Fatal("un recibo escalado no se reintenta sin 'review reset'")
	}
	if err := Reset(root); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(root, DetectCommands(root), opts); err != nil {
		t.Fatalf("tras reset empieza de nuevo: %v", err)
	}
}

func TestInspectReadsFrozenCandidate(t *testing.T) {
	root := highRepo(t)
	r, _ := Run(root, DetectCommands(root), RunOptions{})
	write(t, root, "internal/auth/login.go", "package auth // cambiado después\n")
	out, err := Inspect(root, r, "object", "internal/auth/login.go")
	if err != nil || out != "package auth\n" {
		t.Fatalf("el revisor ve el candidato congelado, no el árbol vivo: %q %v", out, err)
	}
	if ns, _ := Inspect(root, r, "name-status", ""); !strings.Contains(ns, "internal/auth/login.go") {
		t.Fatalf("name-status: %q", ns)
	}
	if _, err := Inspect(root, r, "borrar", ""); err == nil {
		t.Fatal("operación desconocida")
	}
}

func TestModeOffWins(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	if on, src := Mode(root, home); !on || src != "default" {
		t.Fatal("por defecto ON")
	}
	SetMode(root, home, "project", false)
	if on, src := Mode(root, home); on || src != "project" {
		t.Fatalf("OFF de proyecto: %v %s", on, src)
	}
	SetMode(root, home, "project", true)
	SetMode(root, home, "global", false)
	if on, src := Mode(root, home); on || src != "global" {
		t.Fatalf("OFF global: %v %s", on, src)
	}
}

func TestAssessRepoWithoutCommitsAndWithChange(t *testing.T) {
	root := t.TempDir()
	gitIn(t, root, "init", "-q")
	write(t, root, "internal/auth/login.go", "package auth\n")
	if a := Assess(root, ""); a.Tier != TierHigh || !a.ReviewDue {
		t.Fatalf("sin commits, archivo nuevo en auth/ => high: %+v", a)
	}
	root2 := t.TempDir()
	gitIn(t, root2, "init", "-q")
	write(t, root2, "README.md", "hola\n")
	if a := Assess(root2, ""); a.Tier != TierPassive || a.ReviewDue {
		t.Fatalf("solo docs => passive: %+v", a)
	}
}
