package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"v-ia/internal/review"
)

// Flujo completo por la CLI: petición, bloqueante, refutación y recibo válido.
func TestReviewRequestAndRefuteCLI(t *testing.T) {
	root := t.TempDir()
	t.Setenv("V_IA_HOME", t.TempDir())
	t.Chdir(root)
	if out, err := exec.Command("git", "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	write := func(name, body string) {
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".v-ia/checks.txt", "true\n")
	write("auth.go", "package a\n")
	ok := func(args ...string) {
		t.Helper()
		if code := run(args); code != 0 {
			t.Fatalf("%v salió con %d", args, code)
		}
	}
	ok("review", "check", "--request", "valida el token", "--lenses", "risk", "--lenses-reason", "auth")
	r, err := review.Load(root)
	if err != nil || r.Request != "valida el token" {
		t.Fatalf("%q %v", r.Request, err)
	}
	send := func(file string, v any) {
		b, _ := json.Marshal(v)
		write(".v-ia/"+file, string(b))
	}
	send("in.json", review.ReviewInput{CandidateID: r.CandidateID, Lens: "risk", Reviewer: "a", Inspection: review.Inspection{Status: "complete", Paths: r.Paths},
		Findings: []review.Finding{{Location: "auth.go:1", Severity: "BLOCKER", Claim: "x", CausalDisposition: "introduced", ProofRefs: []string{"p"}}}})
	ok("review", "ack", "--input", ".v-ia/in.json")
	if run([]string{"review", "verify"}) == 0 {
		t.Fatal("con bloqueante no debe verificar")
	}
	send("ref.json", review.RefuteInput{CandidateID: r.CandidateID, Refuter: "b", Results: []review.RefuteResult{{ID: "risk#1", Verdict: "refuted", Evidence: []string{"no se reproduce"}}}})
	ok("review", "refute", "--input", ".v-ia/ref.json")
	ok("review", "verify")
}
