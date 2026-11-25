package review

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// maxRequest limita la petición congelada: si no cabe se rechaza, nunca se recorta.
const maxRequest = 8000

// Blocking es un hallazgo que hoy impide aprobar, con el ID con el que se refuta.
type Blocking struct {
	ID        string   `json:"id"`
	Lens      string   `json:"lens"`
	Location  string   `json:"location"`
	Claim     string   `json:"claim"`
	ProofRefs []string `json:"proof_refs"`
}

type RefuteResult struct {
	ID       string   `json:"id"`
	Verdict  string   `json:"verdict"` // refuted | upheld (ante la duda, upheld)
	Evidence []string `json:"evidence"`
}

// RefuteInput es lo que entrega el refutador (JSON estricto).
type RefuteInput struct {
	CandidateID string         `json:"candidate_id"`
	Refuter     string         `json:"refuter"`
	Results     []RefuteResult `json:"results"`
}

type Refutation struct {
	Refuter string         `json:"refuter"`
	Results []RefuteResult `json:"results"`
	At      string         `json:"at"`
}

// Blocking lista los bloqueantes no refutados. El ID es <lente>#<n>.
func (r Receipt) Blocking() []Blocking {
	refuted := map[string]bool{}
	if r.Refutation != nil {
		for _, x := range r.Refutation.Results {
			refuted[x.ID] = x.Verdict == "refuted"
		}
	}
	var out []Blocking
	for _, lr := range r.Reviews {
		n := 0
		for _, f := range lr.Findings {
			if !isBlocking(f) {
				continue
			}
			n++
			if id := fmt.Sprintf("%s#%d", lr.Lens, n); !refuted[id] {
				out = append(out, Blocking{id, lr.Lens, f.Location, f.Claim, f.ProofRefs})
			}
		}
	}
	return out
}

// Refute admite la única refutación de un candidato bloqueado. Debe responder a todos los
// bloqueantes y venir de alguien distinto de quien los reportó.
func Refute(root string, in RefuteInput) (Receipt, error) {
	r, err := load(root)
	if err != nil {
		return r, err
	}
	if r.State != StateCorrectionRequired && r.State != StateEscalated {
		return r, errors.New("no hay bloqueantes que refutar (estado " + r.State + ")")
	}
	if r.Refutation != nil {
		return r, errors.New("este candidato ya tuvo su única refutación")
	}
	cur, err := currentID(root, r.Base)
	if err != nil {
		return r, err
	}
	if cur != r.CandidateID || in.CandidateID != r.CandidateID {
		return r, errors.New("el árbol cambió (o candidate_id no coincide) desde 'review check'")
	}
	who := strings.TrimSpace(in.Refuter)
	if who == "" {
		return r, errors.New("falta 'refuter'")
	}
	pending := map[string]string{} // id -> revisor que lo reportó
	for _, b := range r.Blocking() {
		for _, lr := range r.Reviews {
			if lr.Lens == b.Lens {
				pending[b.ID] = lr.Reviewer
			}
		}
	}
	total := len(pending)
	for _, x := range in.Results {
		reviewer, ok := pending[x.ID]
		switch {
		case !ok:
			return r, fmt.Errorf("%q no es un bloqueante pendiente (o está repetido)", x.ID)
		case strings.EqualFold(strings.TrimSpace(reviewer), who):
			return r, errors.New("quien reportó un hallazgo no puede refutarlo")
		case x.Verdict != "refuted" && x.Verdict != "upheld":
			return r, fmt.Errorf("%s: verdict debe ser refuted o upheld", x.ID)
		case x.Verdict == "refuted" && strings.TrimSpace(strings.Join(x.Evidence, "")) == "":
			return r, fmt.Errorf("%s: refutar exige evidence", x.ID)
		}
		delete(pending, x.ID)
	}
	if total == 0 || len(pending) > 0 {
		return r, fmt.Errorf("la refutación debe responder a los %d bloqueantes; faltan %d", total, len(pending))
	}
	r.Refutation = &Refutation{Refuter: who, Results: in.Results, At: time.Now().UTC().Format(time.RFC3339)}
	if len(r.Blocking()) == 0 {
		r.State = StateApproved
	}
	return r, save(root, r)
}
