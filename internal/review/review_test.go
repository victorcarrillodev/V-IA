package review

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func setup(t *testing.T, cmds string) string {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a.txt"), []byte("1"), 0o644)
	os.MkdirAll(filepath.Join(root, ".v-ia"), 0o755)
	os.WriteFile(filepath.Join(root, ".v-ia", "checks.txt"), []byte(cmds), 0o644)
	return root
}

func TestPassThenByteChangeInvalidates(t *testing.T) {
	root := setup(t, "true\n")
	r, err := Run(root, DetectCommands(root), time.Minute)
	if err != nil || r.Result != "pass" {
		t.Fatalf("esperaba pass: %v %+v", err, r)
	}
	if err := Verify(root, false); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "a.txt"), []byte("2"), 0o644)
	if Verify(root, false) == nil {
		t.Fatal("cambiar un byte debe invalidar el recibo")
	}
}

func TestFailingCheckGivesFail(t *testing.T) {
	root := setup(t, "false\n")
	r, _ := Run(root, DetectCommands(root), time.Minute)
	if r.Result != "fail" || Verify(root, false) == nil {
		t.Fatal("un check fallido no debe dar recibo válido")
	}
}

func TestNoChecksIsError(t *testing.T) {
	if _, err := Run(t.TempDir(), nil, time.Minute); err == nil {
		t.Fatal("sin comprobaciones no puede haber recibo")
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]int
		diff  string
		tier  string
	}{
		{"docs", map[string]int{"README.md": 40}, "", TierPassive},
		{"pequeño con test", map[string]int{"a.go": 10, "a_test.go": 10}, "", TierPassive},
		{"sin tests y grande", map[string]int{"a.go": 60}, "", TierMedium},
		{"muy amplio", map[string]int{"a.go": 200, "a_test.go": 5}, "", TierMedium},
		{"ruta de auth", map[string]int{"internal/auth/login.go": 5, "internal/auth/login_test.go": 5}, "", TierHigh},
		{"migración sql", map[string]int{"db/001.sql": 3}, "", TierHigh},
		{"contenido sensible", map[string]int{"a.go": 3, "a_test.go": 3}, "+\tgo func() {", TierHigh},
		{"workflow", map[string]int{".github/workflows/ci.yml": 3}, "", TierHigh},
	}
	for _, c := range cases {
		if a := classify(c.files, c.diff); a.Tier != c.tier {
			t.Errorf("%s: %s (%v), quería %s", c.name, a.Tier, a.Reasons, c.tier)
		}
	}
}

func TestAssessOutsideGitIsUnknown(t *testing.T) {
	a := Assess(t.TempDir(), "")
	if a.Tier != TierUnknown {
		t.Fatalf("sin git no debe asumir riesgo bajo: %+v", a)
	}
	if len(a.Reasons) != 1 || len(a.Reasons[0]) > 100 {
		t.Fatalf("el motivo debe ser una frase corta, no la ayuda de git: %q", a.Reasons)
	}
}

func TestHighRiskNeedsAck(t *testing.T) {
	root := setup(t, "true\n")
	r, _ := Run(root, DetectCommands(root), time.Minute)
	r.Risk.ReviewDue = true
	if err := save(root, r); err != nil {
		t.Fatal(err)
	}
	if Verify(root, true) == nil {
		t.Fatal("riesgo alto sin acuse debe fallar")
	}
	if Verify(root, false) != nil {
		t.Fatal("con RDD apagado no exige acuse")
	}
	if err := RecordAck(root, "changes", "rev", "arregla X"); err != nil {
		t.Fatal(err)
	}
	if Verify(root, true) == nil {
		t.Fatal("un 'changes' no es aprobación")
	}
	RecordAck(root, "approve", "rev", "")
	if err := Verify(root, true); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "a.txt"), []byte("cambio"), 0o644)
	if Verify(root, true) == nil {
		t.Fatal("cambiar el árbol invalida recibo y acuse")
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
	run := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = root
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run("init", "-q")
	os.MkdirAll(filepath.Join(root, "internal", "auth"), 0o755)
	os.WriteFile(filepath.Join(root, "internal", "auth", "login.go"), []byte("package auth\n"), 0o644)
	if a := Assess(root, ""); a.Tier != TierHigh || !a.ReviewDue {
		t.Fatalf("sin commits, archivo nuevo en auth/ => high: %+v", a)
	}
	run("add", "-A")
	run("-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "init")
	if a := Assess(root, ""); a.Tier != TierPassive {
		t.Fatalf("sin cambios => passive: %+v", a)
	}
	os.WriteFile(filepath.Join(root, "internal", "auth", "login.go"), []byte("package auth\n// x\n"), 0o644)
	if a := Assess(root, ""); a.Tier != TierHigh {
		t.Fatalf("editar auth/ => high: %+v", a)
	}
}
