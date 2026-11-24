package review

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"v-ia/internal/guard"
)

// emptyTree es el árbol vacío de git: la base de un repositorio sin commits.
const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// Candidate es el cambio congelado que se revisa: dos árboles git inmutables (base y candidato)
// y la lista de rutas cambiadas. Los revisores leen estos árboles, no el árbol de trabajo vivo.
type Candidate struct {
	BaseTree      string   `json:"base_tree"`
	CandidateTree string   `json:"candidate_tree"`
	Paths         []string `json:"paths"`
	ID            string   `json:"id"`
}

func gitEnv(root string, env []string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func isRepo(root string) bool {
	out, err := git(root, "rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(out) == "true"
}

// baseTree resuelve el árbol de la referencia base (por defecto HEAD; sin commits, el árbol vacío).
func baseTree(root, base string) (string, error) {
	if base == "" {
		base = "HEAD"
	}
	if base == "HEAD" {
		if _, err := git(root, "rev-parse", "--verify", "-q", "HEAD"); err != nil {
			return emptyTree, nil
		}
	}
	out, err := git(root, "rev-parse", base+"^{tree}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// TakeCandidate congela el estado actual del árbol de trabajo (archivos con seguimiento y nuevos no
// ignorados) como un árbol git, sin tocar el índice ni crear commits. Falla si no es un repositorio.
func TakeCandidate(root, base string) (Candidate, error) {
	if !isRepo(root) {
		return Candidate{}, fmt.Errorf("no es un repositorio git")
	}
	bt, err := baseTree(root, base)
	if err != nil {
		return Candidate{}, err
	}
	tmp, err := os.MkdirTemp("", "v-ia-index")
	if err != nil {
		return Candidate{}, err
	}
	defer os.RemoveAll(tmp)
	env := []string{"GIT_INDEX_FILE=" + filepath.Join(tmp, "index")}
	if bt != emptyTree {
		if _, err := gitEnv(root, env, "read-tree", bt); err != nil {
			return Candidate{}, err
		}
	}
	if _, err := gitEnv(root, env, "add", "-A", "--", ".", ":(exclude).v-ia", ":(exclude).atl", ":(exclude).serena"); err != nil {
		return Candidate{}, err
	}
	ct, err := gitEnv(root, env, "write-tree")
	if err != nil {
		return Candidate{}, err
	}
	names, err := git(root, "diff-tree", "-r", "--name-only", "--no-renames", bt, ct)
	if err != nil {
		return Candidate{}, err
	}
	var paths []string
	for _, p := range strings.Split(strings.TrimSpace(names), "\n") {
		if p != "" {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	h := sha256.Sum256([]byte("v-ia.candidate/v1\x00" + bt + "\x00" + ct))
	return Candidate{BaseTree: bt, CandidateTree: ct, Paths: paths, ID: hex.EncodeToString(h[:])}, nil
}

// currentID calcula el identificador del árbol actual: el del candidato git si hay repo y, si no,
// la huella de archivos (así un directorio sin git sigue pudiendo recibir un recibo).
func currentID(root, base string) (string, error) {
	if c, err := TakeCandidate(root, base); err == nil {
		return c.ID, nil
	}
	snap, err := guard.Take(root)
	if err != nil {
		return "", err
	}
	return guard.ID(snap), nil
}

// Inspect lee del candidato congelado (no del árbol vivo). op: name-status | numstat | stat | patch | object.
func Inspect(root string, r Receipt, op, path string) (string, error) {
	if r.CandidateTree == "" {
		return "", fmt.Errorf("el recibo no tiene candidato congelado (¿no es un repositorio git?)")
	}
	for _, t := range []string{r.BaseTree, r.CandidateTree} {
		if _, err := git(root, "cat-file", "-e", t); err != nil {
			return "", fmt.Errorf("el árbol %.12s ya no existe en el repositorio (podado): vuelve a ejecutar 'v-ia review check'", t)
		}
	}
	switch op {
	case "name-status", "numstat", "stat":
		return git(root, "diff-tree", "-r", "--no-renames", "--"+op, r.BaseTree, r.CandidateTree)
	case "patch":
		args := []string{"diff-tree", "-r", "-p", "--no-renames", r.BaseTree, r.CandidateTree}
		if path != "" {
			args = append(args, "--", path)
		}
		return git(root, args...)
	case "object":
		if path == "" {
			return "", fmt.Errorf("object requiere --path")
		}
		return git(root, "show", r.CandidateTree+":"+path)
	}
	return "", fmt.Errorf("operación desconocida %q (name-status | numstat | stat | patch | object)", op)
}
