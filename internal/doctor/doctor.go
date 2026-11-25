// Package doctor hace el diagnóstico de solo lectura (sin UI).
package doctor

import (
	"fmt"
	"os"
	"strings"

	"v-ia/internal/adapters"
	"v-ia/internal/core"
	"v-ia/internal/extras"
	"v-ia/internal/install"
	"v-ia/internal/lifecycle"
	"v-ia/internal/settings"
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
		// Los principales se informan siempre; el resto, solo si están presentes.
		if !a.Detected() && !adapters.IsPrimary(a.Name) {
			continue
		}
		if a.Detected() {
			rs = append(rs, Result{Pass, a.Name + ": detectado"})
		} else {
			rs = append(rs, Result{Warn, a.Name + ": no encontrado en PATH"})
		}
		switch {
		case a.HooksVerified:
			rs = append(rs, Result{Pass, a.Name + ": ODD se hace cumplir por bloqueo (hooks)"})
		case a.Hooks:
			rs = append(rs, Result{Warn, a.Name + ": hooks sin verificar con el agente real; trata ODD como recomendación"})
		default:
			rs = append(rs, Result{Warn, a.Name + ": solo instrucciones; sin hooks ODD es una recomendación"})
		}
		if a.Name == "opencode" && a.Detected() {
			switch m, raw, ok := adapters.OpenCodeMajor(); {
			case !ok:
				rs = append(rs, Result{Warn, "opencode: no se pudo determinar la versión (`opencode --version`); el plugin solo está verificado con OpenCode 1.x"})
			case m != adapters.OpenCodeVerifiedMajor:
				rs = append(rs, Result{Warn, fmt.Sprintf("opencode %s: el plugin y los skills solo están verificados con OpenCode %d.x; con otra versión mayor pueden no cargarse", strings.TrimSpace(raw), adapters.OpenCodeVerifiedMajor)})
			default:
				rs = append(rs, Result{Pass, "opencode " + strings.TrimSpace(raw) + ": versión mayor verificada"})
			}
		}
		rs = append(rs, files(home, a)...)
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
	if r, ok := install.Load(home); ok {
		rs = append(rs, Result{Pass, "instalación registrada: preset " + r.Preset + ", agentes " + strings.Join(r.Agents, ", ")})
	} else {
		rs = append(rs, Result{Warn, "sin registro de instalación (~/.v-ia/install.json) → v-ia install"})
	}
	if b, err := os.ReadFile(adapters.SettingsPath(home)); err == nil && settings.Installed(string(b)) {
		rs = append(rs, Result{Pass, "hooks ODD (bloqueo de edición/entrega/cierre): instalados"})
	} else {
		rs = append(rs, Result{Warn, "hooks ODD no instalados: sin ellos ODD es solo una recomendación → v-ia install"})
	}
	for _, it := range extras.Status(home) {
		if it.OK {
			rs = append(rs, Result{Pass, it.Msg})
		} else {
			rs = append(rs, Result{Warn, it.Msg})
		}
	}
	return rs
}

// files comprueba las skills, el subagente y el plugin que V-IA gestiona para un agente.
func files(home string, a adapters.Adapter) []Result {
	p, err := lifecycle.Install(lifecycle.Options{Home: home, Agents: []string{a.Name}, Hooks: true, Permissions: true})
	if err != nil {
		return []Result{{Fail, a.Name + ": no pude planificar los archivos gestionados: " + err.Error()}}
	}
	var ok, missing, edited int
	for _, x := range p.Actions {
		if x.Path == adapters.SettingsPath(home) || x.Path == adapters.HookConfigPath(home, a.Name) || isBlockTarget(home, a, x.Path) {
			continue // los bloques y settings se informan aparte
		}
		switch x.Status {
		case core.Unchanged:
			ok++
		case core.Conflict:
			edited++
		default:
			missing++
		}
	}
	switch {
	case edited > 0:
		return []Result{{Fail, fmt.Sprintf("%s: %d archivos gestionados editados a mano (v-ia install --force)", a.Name, edited)}}
	case missing > 0 && ok == 0:
		return []Result{{Warn, a.Name + ": skills y plugin sin instalar → v-ia install --agent " + a.Name}}
	case missing > 0:
		return []Result{{Warn, fmt.Sprintf("%s: %d de %d archivos gestionados desactualizados → v-ia sync", a.Name, missing, missing+ok)}}
	}
	return []Result{{Pass, fmt.Sprintf("%s: %d archivos gestionados (skills/plugin) al día", a.Name, ok)}}
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

func isBlockTarget(home string, a adapters.Adapter, path string) bool {
	for _, t := range a.Targets(home) {
		if t == path {
			return true
		}
	}
	return false
}
