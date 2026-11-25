package review

import (
	"strings"
	"testing"
)

func refutation(r Receipt, verdict string) RefuteInput {
	return RefuteInput{CandidateID: r.CandidateID, Refuter: "otro", Results: []RefuteResult{{ID: "risk#1", Verdict: verdict, Evidence: []string{"go test ./... pasa"}}}}
}

func TestRefutedBlockerApproves(t *testing.T) {
	root, r := blocked(t)
	if b := Status(root).Blocking; len(b) != 1 || b[0].ID != "risk#1" {
		t.Fatalf("bloqueantes: %+v", b)
	}
	r, err := Refute(root, refutation(r, "refuted"))
	if err != nil || r.State != StateApproved {
		t.Fatalf("estado=%s err=%v", r.State, err)
	}
	if err := Verify(root, true); err != nil {
		t.Fatal(err)
	}
}

func TestUpheldBlockerStillBlocks(t *testing.T) {
	root, r := blocked(t)
	r, err := Refute(root, refutation(r, "upheld"))
	if err != nil || r.State != StateCorrectionRequired {
		t.Fatalf("estado=%s err=%v", r.State, err)
	}
	if _, err := Refute(root, refutation(r, "refuted")); err == nil {
		t.Error("solo hay una refutación por candidato")
	}
}

func TestRefutationRejected(t *testing.T) {
	for name, mutate := range map[string]func(*RefuteInput){
		"mismo revisor":  func(in *RefuteInput) { in.Refuter = "x" },
		"sin prueba":     func(in *RefuteInput) { in.Results[0].Evidence = nil },
		"falta hallazgo": func(in *RefuteInput) { in.Results = nil },
		"id desconocido": func(in *RefuteInput) { in.Results[0].ID = "risk#9" },
		"veredicto raro": func(in *RefuteInput) { in.Results[0].Verdict = "quizá" },
		"otro candidato": func(in *RefuteInput) { in.CandidateID = "abc" },
	} {
		root, r := blocked(t)
		in := refutation(r, "refuted")
		mutate(&in)
		if _, err := Refute(root, in); err == nil {
			t.Errorf("%s: debió rechazarse", name)
		}
		if st := Status(root); st.Detail != StateCorrectionRequired {
			t.Errorf("%s: estado %s", name, st.Detail)
		}
	}
}

func TestRequestFrozenInReceipt(t *testing.T) {
	root := repo(t)
	put(t, root, "auth.go", "package a\n\nfunc A() { _ = 1 }\n")
	o := onlyRisk
	o.Request = "valida el token\ny devuelve 401"
	r, err := Run(root, []string{"true"}, o)
	if err != nil {
		t.Fatal(err)
	}
	if got := Status(root).Request; got != o.Request {
		t.Fatalf("petición: %q", got)
	}
	if _, err := RecordReview(root, input(r, "BLOCKER")); err != nil {
		t.Fatal(err)
	}
	put(t, root, "auth.go", "package a\n\nfunc A() { _ = 2 }\n")
	o.Request = "otra cosa"
	if _, err := Run(root, []string{"true"}, o); err == nil {
		t.Error("la corrección no puede cambiar la petición")
	}
	o.Request = ""
	r, err = Run(root, []string{"true"}, o)
	if err != nil || !strings.Contains(r.Request, "401") {
		t.Fatalf("debe conservarse: %q %v", r.Request, err)
	}
	o.Request = strings.Repeat("x", 8001)
	if _, err := Run(root, []string{"true"}, o); err == nil {
		t.Error("una petición demasiado larga se rechaza, no se recorta")
	}
}
