package core

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Target es un archivo y el bloque que V-IA debe gestionar en él.
type Target struct {
	Path  string
	Block Block
}

// Action es un cambio planificado sobre un archivo.
type Action struct {
	Path    string
	Status  Status
	Existed bool
	Old     string
	New     string
}

// Plan es la lista de acciones; construirlo no escribe nada (dry-run).
type Plan struct {
	Actions []Action
	// Warnings son avisos no fatales (p. ej. un agente cuya configuración no se pudo fusionar y se omitió).
	Warnings []string
}

// BuildPlan calcula qué cambiaría en disco sin tocarlo.
func BuildPlan(targets []Target, force bool) (Plan, error) {
	var p Plan
	idx := map[string]int{} // ruta -> posición de su acción: varios bloques en un archivo se acumulan
	for _, t := range targets {
		if i, ok := idx[t.Path]; ok {
			a := &p.Actions[i]
			nw, st := Upsert(a.New, true, t.Block, force)
			a.New = nw
			switch {
			case st == Conflict:
				a.Status = Conflict
			case st != Unchanged && a.Status == Unchanged:
				a.Status = st
			}
			continue
		}
		data, err := os.ReadFile(t.Path)
		exists := err == nil
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return Plan{}, err
		}
		old := string(data)
		nw, st := Upsert(old, exists, t.Block, force)
		idx[t.Path] = len(p.Actions)
		p.Actions = append(p.Actions, Action{Path: t.Path, Status: st, Existed: exists, Old: old, New: nw})
	}
	return p, nil
}

// Changes devuelve solo las acciones que escriben algo.
func (p Plan) Changes() []Action {
	var out []Action
	for _, a := range p.Actions {
		if a.Status == Create || a.Status == Add || a.Status == Update || a.Status == Delete {
			out = append(out, a)
		}
	}
	return out
}

// Conflicts devuelve las acciones bloqueadas por edición manual.
func (p Plan) Conflicts() []Action {
	var out []Action
	for _, a := range p.Actions {
		if a.Status == Conflict {
			out = append(out, a)
		}
	}
	return out
}

// Diff muestra el diff por líneas (LCS) de una acción.
func (a Action) Diff() string {
	x, y := splitLines(a.Old), splitLines(a.New)
	n, m := len(x), len(y)
	l := make([][]int, n+1)
	for i := range l {
		l[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if x[i] == y[j] {
				l[i][j] = l[i+1][j+1] + 1
			} else {
				l[i][j] = max(l[i+1][j], l[i][j+1])
			}
		}
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s (%s)\n", a.Path, a.Status)
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case x[i] == y[j]:
			i++
			j++
		case l[i+1][j] >= l[i][j+1]:
			fmt.Fprintf(&sb, "-%s\n", x[i])
			i++
		default:
			fmt.Fprintf(&sb, "+%s\n", y[j])
			j++
		}
	}
	for ; i < n; i++ {
		fmt.Fprintf(&sb, "-%s\n", x[i])
	}
	for ; j < m; j++ {
		fmt.Fprintf(&sb, "+%s\n", y[j])
	}
	return sb.String()
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}
