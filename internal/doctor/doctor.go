// Package doctor hace el diagnóstico de solo lectura (sin UI).
package doctor

import (
	"os/exec"

	"v-ia/internal/adapters"
	"v-ia/internal/core"
)

type Level int

const (
	Pass Level = iota
	Warn
	Fail
)

func (l Level) Symbol() string { return [...]string{"✓", "!", "✗"}[l] }

type Result struct {
	Level Level
	Msg   string
}

// Run diagnostica los adaptadores y Engram. No escribe nada.
func Run(home string) []Result {
	var rs []Result
	for _, a := range adapters.All {
		if a.Detected() {
			rs = append(rs, Result{Pass, a.Name + ": detectado"})
		} else {
			rs = append(rs, Result{Warn, a.Name + ": no encontrado en PATH"})
		}
		ts, _ := adapters.Targets(home, []string{a.Name})
		p, _ := core.BuildPlan(ts, false)
		for _, x := range p.Actions {
			switch x.Status {
			case core.Unchanged:
				rs = append(rs, Result{Pass, x.Path + ": bloque ODD al día"})
			case core.Conflict:
				rs = append(rs, Result{Fail, x.Path + ": bloque editado a mano (v-ia install --force)"})
			default:
				rs = append(rs, Result{Warn, x.Path + ": bloque ODD pendiente (" + x.Status.String() + ") → v-ia install"})
			}
		}
	}
	if _, err := exec.LookPath("engram"); err == nil {
		rs = append(rs, Result{Pass, "engram: instalado"})
	} else {
		rs = append(rs, Result{Warn, "engram: no encontrado (respaldo .v-ia/memory.md)"})
	}
	return rs
}

// Failed indica si algún resultado es Fail.
func Failed(rs []Result) bool {
	for _, r := range rs {
		if r.Level == Fail {
			return true
		}
	}
	return false
}
