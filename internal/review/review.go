// Package review ejecuta las comprobaciones del proyecto y emite un recibo
// ligado al contenido exacto del árbol (candidate_id).
package review

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"v-ia/internal/guard"
)

type Check struct {
	Cmd    string `json:"cmd"`
	Exit   int    `json:"exit"`
	Output string `json:"output_tail"`
}

// Ack es el veredicto de un revisor independiente, ligado al candidato exacto.
type Ack struct {
	CandidateID string `json:"candidate_id"`
	Verdict     string `json:"verdict"` // approve | changes
	Reviewer    string `json:"reviewer"`
	Note        string `json:"note,omitempty"`
	At          string `json:"at"`
}

type Receipt struct {
	CandidateID string     `json:"candidate_id"`
	Created     string     `json:"created"`
	Result      string     `json:"result"` // pass | fail
	Checks      []Check    `json:"checks"`
	Risk        Assessment `json:"risk"`
	Review      *Ack       `json:"review,omitempty"`
}

func receiptPath(root string) string { return filepath.Join(root, ".v-ia", "receipt.json") }

// DetectCommands decide qué comprobar: .v-ia/checks.txt manda; si no, se infiere.
func DetectCommands(root string) []string {
	if f, err := os.Open(filepath.Join(root, ".v-ia", "checks.txt")); err == nil {
		defer f.Close()
		var out []string
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			if l := strings.TrimSpace(sc.Text()); l != "" && !strings.HasPrefix(l, "#") {
				out = append(out, l)
			}
		}
		return out
	}
	var out []string
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
		out = append(out, "go build ./...", "go vet ./...", "go test ./...")
	}
	if b, err := os.ReadFile(filepath.Join(root, "package.json")); err == nil && strings.Contains(string(b), `"test"`) {
		out = append(out, "npm test --silent")
	}
	return out
}

func tail(s string, n int) string {
	if len(s) > n {
		return "…" + s[len(s)-n:]
	}
	return s
}

// Run ejecuta las comprobaciones y guarda el recibo. Sin comprobaciones => error:
// un recibo vacío no demuestra nada.
func Run(root string, cmds []string, timeout time.Duration) (Receipt, error) {
	if len(cmds) == 0 {
		return Receipt{}, errors.New("no hay comprobaciones: añade comandos en .v-ia/checks.txt (uno por línea)")
	}
	r := Receipt{Result: "pass"}
	for _, c := range cmds {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		cmd := exec.CommandContext(ctx, "sh", "-c", c)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		cancel()
		ck := Check{Cmd: c, Output: tail(string(out), 1500)}
		if err != nil {
			ck.Exit = 1
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				ck.Exit = ee.ExitCode()
			}
			r.Result = "fail"
		}
		r.Checks = append(r.Checks, ck)
	}
	snap, err := guard.Take(root)
	if err != nil {
		return r, err
	}
	r.CandidateID = guard.ID(snap)
	r.Risk = Assess(root, "")
	r.Created = time.Now().UTC().Format(time.RFC3339)
	return r, save(root, r)
}

func save(root string, r Receipt) error {
	b, _ := json.MarshalIndent(r, "", "  ")
	if err := os.MkdirAll(filepath.Dir(receiptPath(root)), 0o755); err != nil {
		return err
	}
	return os.WriteFile(receiptPath(root), b, 0o644)
}

func load(root string) (Receipt, error) {
	var r Receipt
	raw, err := os.ReadFile(receiptPath(root))
	if err != nil {
		return r, errors.New("no hay recibo: ejecuta 'v-ia review check'")
	}
	return r, json.Unmarshal(raw, &r)
}

// RecordAck guarda el veredicto del revisor si el árbol sigue siendo el candidato del recibo.
func RecordAck(root, verdict, reviewer, note string) error {
	if verdict != "approve" && verdict != "changes" {
		return errors.New("veredicto inválido: approve | changes")
	}
	r, err := load(root)
	if err != nil {
		return err
	}
	snap, err := guard.Take(root)
	if err != nil {
		return err
	}
	if guard.ID(snap) != r.CandidateID {
		return errors.New("el árbol cambió desde 'review check': vuelve a ejecutarlo antes de revisar")
	}
	r.Review = &Ack{CandidateID: r.CandidateID, Verdict: verdict, Reviewer: reviewer, Note: note, At: time.Now().UTC().Format(time.RFC3339)}
	return save(root, r)
}

// Verify confirma que hay un recibo pass para el árbol actual. Con requireAck
// (riesgo alto y RDD activo) exige además el visto bueno de un revisor sobre ese mismo candidato.
func Verify(root string, requireAck bool) error {
	r, err := load(root)
	if err != nil {
		return err
	}
	if r.Result != "pass" {
		return errors.New("el último recibo es fail")
	}
	snap, err := guard.Take(root)
	if err != nil {
		return err
	}
	if guard.ID(snap) != r.CandidateID {
		return errors.New("el recibo es de otra versión: el árbol cambió desde la revisión")
	}
	if requireAck && r.Risk.ReviewDue {
		if r.Review == nil || r.Review.CandidateID != r.CandidateID {
			return errors.New("riesgo ALTO: falta la revisión independiente (subagente v-ia-reviewer → v-ia review ack)")
		}
		if r.Review.Verdict != "approve" {
			return errors.New("el revisor pidió cambios: " + r.Review.Note)
		}
	}
	return nil
}

// Load devuelve el último recibo guardado (para mostrarlo; no lo valida).
func Load(root string) (Receipt, error) { return load(root) }
