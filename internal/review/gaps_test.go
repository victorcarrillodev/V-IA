package review

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func repo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	run := func(a ...string) {
		c := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t"}, a...)...)
		c.Dir = root
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", a, err, out)
		}
	}
	run("init", "-q")
	put(t, root, "a.go", "package a\n")
	run("add", "-A")
	run("commit", "-qm", "1")
	put(t, root, "auth.go", "package a\n\nfunc A() {}\n")
	run("add", "-A")
	run("commit", "-qm", "2")
	return root
}

func put(t *testing.T, root, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func input(r Receipt, severity string) ReviewInput {
	in := ReviewInput{CandidateID: r.CandidateID, Lens: "risk", Reviewer: "x", Inspection: Inspection{Status: "complete", Paths: r.Paths}}
	if severity != "" {
		in.Findings = []Finding{{Location: "auth.go:3", Severity: severity, Claim: "fallo", CausalDisposition: "introduced", ProofRefs: []string{"p"}}}
	}
	return in
}

var onlyRisk = RunOptions{Lenses: []string{"risk"}, LensReason: "auth"}

func blocked(t *testing.T) (string, Receipt) {
	t.Helper()
	root := repo(t)
	put(t, root, "auth.go", "package a\n\nfunc A() { _ = 1 }\n")
	r, err := Run(root, []string{"true"}, onlyRisk)
	if err != nil {
		t.Fatal(err)
	}
	r, err = RecordReview(root, input(r, "BLOCKER"))
	if err != nil || r.State != StateCorrectionRequired {
		t.Fatalf("estado=%s err=%v", r.State, err)
	}
	return root, r
}

// Un recibo emitido contra --base sigue siendo válido mientras nada cambie.
func TestReceiptWithBaseStaysCurrent(t *testing.T) {
	root := repo(t)
	o := onlyRisk
	o.Base = "HEAD~1"
	r, err := Run(root, []string{"true"}, o)
	if err != nil {
		t.Fatal(err)
	}
	if st := Status(root); st.Detail == "candidato_cambiado" {
		t.Fatalf("status: %+v", st)
	}
	if _, err := RecordReview(root, input(r, "")); err != nil {
		t.Fatalf("ack: %v", err)
	}
	if err := Verify(root, true); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

// Un bloqueante no se anula reenviando la revisión ni repitiendo check sin corregir.
func TestBlockerCannotBeOverwritten(t *testing.T) {
	root, r := blocked(t)
	if _, err := RecordReview(root, input(r, "")); err == nil {
		t.Error("se aceptó un segundo veredicto para la misma lente")
	}
	if _, err := Run(root, []string{"true"}, onlyRisk); err == nil {
		t.Error("check sin cambiar el código debe rechazarse")
	}
}

// Una corrección que falla las pruebas no devuelve el presupuesto de correcciones.
func TestFailedCorrectionKeepsAttempt(t *testing.T) {
	root, _ := blocked(t)
	put(t, root, "auth.go", "package a\n\nfunc A() { _ = 2 }\n")
	if _, err := Run(root, []string{"false"}, onlyRisk); err != nil {
		t.Fatal(err)
	}
	r, err := Run(root, []string{"true"}, onlyRisk)
	if err != nil {
		t.Fatal(err)
	}
	r, err = RecordReview(root, input(r, "BLOCKER"))
	if err != nil || r.State != StateEscalated {
		t.Fatalf("estado=%s intento=%d err=%v", r.State, r.Attempt, err)
	}
}

// Volver al código bloqueado tras una corrección fallida no permite revisarlo otra vez.
func TestBlockedCodeCannotComeBack(t *testing.T) {
	root, _ := blocked(t)
	put(t, root, "auth.go", "package a\n\nfunc A() { _ = 2 }\n")
	if _, err := Run(root, []string{"false"}, onlyRisk); err != nil {
		t.Fatal(err)
	}
	put(t, root, "auth.go", "package a\n\nfunc A() { _ = 1 }\n")
	if r, err := Run(root, []string{"true"}, onlyRisk); err == nil {
		t.Errorf("se aceptó el código bloqueado: estado %s", r.State)
	}
}
