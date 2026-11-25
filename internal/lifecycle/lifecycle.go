// Package lifecycle construye los planes de install, sync y uninstall.
// Solo planifica: aplicar (backup, escritura atómica, rollback) es de core.Apply.
package lifecycle

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	corefs "v-ia/core"
	"v-ia/internal/adapters"
	"v-ia/internal/core"
	"v-ia/internal/settings"
)

type Options struct {
	Home        string
	Agents      []string
	Exe         string // comando con el que los hooks invocan a v-ia
	Force       bool
	Hooks       bool
	Permissions bool
	NoSkills    bool // no instalar la biblioteca de skills (preset minimal)
	Persona     string
	Engram      bool // Engram se va a configurar en esta instalación
}

var skillNames = []string{"odd-resume", "odd-verify", "odd-delegate", "rdd-review", "work-unit-commits", "branch-pr", "issue-creation", "chained-pr", "judgment-day", "skill-creator", "comment-writer", "rdd-defect-workflow", "go-testing", "skill-improver", "systemic-issue-triage", "cognitive-doc-design", "skill-registry", "issue-root-resolution"}

// ExeForHooks devuelve la ruta con la que los hooks llamarán a v-ia.
func ExeForHooks() string {
	if exe, err := os.Executable(); err == nil && filepath.Base(exe) == "v-ia" {
		return exe
	}
	return "v-ia"
}

func has(l []string, s string) bool {
	for _, x := range l {
		if strings.TrimSpace(x) == s {
			return true
		}
	}
	return false
}

func read(path string) (string, bool, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	return string(b), err == nil, err
}

type managedFile struct {
	path, content string
	js            bool // la marca de gestión es un comentario JS
}

// files son los archivos completos que V-IA gestiona: la biblioteca de skills en cada agente
// que las admite, el subagente revisor (Claude Code) y el plugin de bloqueo (OpenCode).
func files(o Options) ([]managedFile, error) {
	var out []managedFile
	for _, name := range o.Agents {
		a, ok := adapters.Find(name)
		if !ok || a.SkillsDir == nil {
			continue
		}
		if !o.NoSkills {
			for _, n := range skillNames {
				b, err := corefs.FS.ReadFile("skills/" + n + "/SKILL.md")
				if err != nil {
					return nil, err
				}
				out = append(out, managedFile{path: filepath.Join(a.SkillsDir(o.Home), n, "SKILL.md"), content: string(b)})
			}
		}
		if a.OwnFile != nil {
			path, fm := a.OwnFile(o.Home)
			b, err := a.Block(o.Home, o.Engram)
			if err != nil {
				return nil, err
			}
			content := fm + b.Content
			if pb, ok := adapters.PersonaBlock(o.Persona); ok {
				content += "\n" + pb.Content
			}
			out = append(out, managedFile{path: path, content: content})
		}
		af, err := agentFiles(o, a)
		if err != nil {
			return nil, err
		}
		out = append(out, af...)
		switch name {
		case "opencode":
			if o.Hooks {
				f, err := opencodePlugin(o)
				if err != nil {
					return nil, err
				}
				out = append(out, f)
			}
		}
	}
	return dedupe(out), nil
}

// dedupe quita archivos repetidos (dos agentes pueden compartir la carpeta de skills).
func dedupe(in []managedFile) []managedFile {
	seen := map[string]bool{}
	var out []managedFile
	for _, f := range in {
		if !seen[f.path] {
			seen[f.path] = true
			out = append(out, f)
		}
	}
	return out
}

// opencodePlugin renderiza el plugin que conecta OpenCode con `v-ia hook`.
func opencodePlugin(o Options) (managedFile, error) {
	b, err := corefs.FS.ReadFile("plugins/opencode-v-ia.js")
	if err != nil {
		return managedFile{}, err
	}
	exe, _ := json.Marshal(o.Exe)
	body := strings.ReplaceAll(string(b), "__V_IA_EXE__", string(exe))
	return managedFile{path: filepath.Join(adapters.XDGConfig(o.Home), "opencode", "plugins", "v-ia.js"), content: body, js: true}, nil
}

func fileAction(f managedFile, force bool) (core.Action, error) {
	old, exists, err := read(f.path)
	if err != nil {
		return core.Action{}, err
	}
	up := core.UpsertFile
	if f.js {
		up = core.UpsertFileJS
	}
	nw, st := up(old, exists, f.content, force)
	return core.Action{Path: f.path, Status: st, Existed: exists, Old: old, New: nw}, nil
}

func settingsAction(o Options) (core.Action, error) {
	p := adapters.SettingsPath(o.Home)
	old, exists, err := read(p)
	if err != nil {
		return core.Action{}, err
	}
	nw, err := settings.Merge(old, o.Exe, o.Hooks, o.Permissions)
	if err != nil {
		return core.Action{}, err
	}
	st := core.Update
	switch {
	case !exists:
		st = core.Create
	case nw == old:
		st = core.Unchanged
	}
	return core.Action{Path: p, Status: st, Existed: exists, Old: old, New: nw}, nil
}

// hookAction planifica la fusión de los hooks de V-IA en la configuración de otro agente.
func hookAction(o Options, agent string) (core.Action, error) {
	p := adapters.HookConfigPath(o.Home, agent)
	old, exists, err := read(p)
	if err != nil {
		return core.Action{}, err
	}
	nw, err := settings.HooksMerge(old, o.Exe, settings.Specs[agent])
	if err != nil {
		return core.Action{}, err
	}
	st := core.Update
	switch {
	case !exists:
		st = core.Create
	case nw == old:
		st = core.Unchanged
	}
	return core.Action{Path: p, Status: st, Existed: exists, Old: old, New: nw}, nil
}

// Install planifica bloques de instrucciones, y para Claude Code además hooks, deny-list,
// skills y el subagente revisor.
func Install(o Options) (core.Plan, error) {
	if o.Exe == "" {
		o.Exe = ExeForHooks()
	}
	ts, err := adapters.Targets(o.Home, o.Agents, o.Engram)
	if err != nil {
		return core.Plan{}, err
	}
	if pb, ok := adapters.PersonaBlock(o.Persona); ok {
		for _, t := range ts {
			ts = append(ts, core.Target{Path: t.Path, Block: pb})
		}
	}
	p, err := core.BuildPlan(ts, o.Force)
	if err != nil {
		return core.Plan{}, err
	}
	if has(o.Agents, "claude-code") && (o.Hooks || o.Permissions) {
		a, err := settingsAction(o)
		if err != nil {
			return core.Plan{}, err
		}
		p.Actions = append(p.Actions, a)
	}
	for _, name := range o.Agents {
		if _, ok := settings.Specs[name]; ok && o.Hooks {
			a, err := hookAction(o, name)
			if err != nil {
				// una configuración ajena rota no debe impedir instalar el resto
				p.Warnings = append(p.Warnings, name+": hooks omitidos: "+err.Error())
				continue
			}
			p.Actions = append(p.Actions, a)
		}
	}
	fs, err := files(o)
	if err != nil {
		return core.Plan{}, err
	}
	for _, f := range fs {
		a, err := fileAction(f, o.Force)
		if err != nil {
			return core.Plan{}, err
		}
		p.Actions = append(p.Actions, a)
	}
	return p, nil
}

// Sync actualiza solo lo que ya está instalado: nunca añade componentes nuevos.
func Sync(o Options) (core.Plan, error) {
	full, err := Install(o)
	if err != nil {
		return core.Plan{}, err
	}
	var p core.Plan
	for _, a := range full.Actions {
		switch {
		case a.Status == core.Create || a.Status == core.Add:
			continue // no estaba instalado
		case a.Path == adapters.SettingsPath(o.Home) && !settings.Installed(a.Old):
			continue
		case isAgentHookFile(o, a.Path) && !settings.HooksInstalled(a.Old):
			continue
		}
		p.Actions = append(p.Actions, a)
	}
	return p, nil
}

// Uninstall retira bloques, hooks y archivos gestionados intactos. Lo editado a mano se
// conserva. La deny-list de permisos es decisión del usuario y se deja.
func Uninstall(o Options) (core.Plan, error) {
	var p core.Plan
	ts, err := adapters.Targets(o.Home, o.Agents, o.Engram)
	if err != nil {
		return p, err
	}
	for _, t := range ts {
		old, exists, err := read(t.Path)
		if err != nil {
			return p, err
		}
		if !exists {
			continue
		}
		stripped, hadPersona := core.Remove(old, "persona") // la persona es opcional: se retira siempre
		nw, removed := core.Remove(stripped, t.Block.ID)
		removed = removed || hadPersona
		switch {
		case !removed:
			p.Actions = append(p.Actions, core.Action{Path: t.Path, Status: core.Unchanged, Existed: true, Old: old, New: old})
		case strings.TrimSpace(nw) == "":
			p.Actions = append(p.Actions, core.Action{Path: t.Path, Status: core.Delete, Existed: true, Old: old})
		default:
			p.Actions = append(p.Actions, core.Action{Path: t.Path, Status: core.Update, Existed: true, Old: old, New: nw})
		}
	}
	if has(o.Agents, "claude-code") {
		sp := adapters.SettingsPath(o.Home)
		if old, exists, err := read(sp); err != nil {
			return p, err
		} else if exists && settings.Installed(old) {
			nw, err := settings.Strip(old)
			if err != nil {
				return p, err
			}
			p.Actions = append(p.Actions, core.Action{Path: sp, Status: core.Update, Existed: true, Old: old, New: nw})
		}
	}
	for _, name := range o.Agents {
		if _, ok := settings.Specs[name]; !ok {
			continue
		}
		gp := adapters.HookConfigPath(o.Home, name)
		if old, exists, err := read(gp); err != nil {
			return p, err
		} else if exists && settings.HooksInstalled(old) {
			nw, err := settings.HooksStrip(old)
			if err != nil {
				return p, err
			}
			p.Actions = append(p.Actions, core.Action{Path: gp, Status: core.Update, Existed: true, Old: old, New: nw})
		}
	}
	o.Hooks = true // al desinstalar se retira también el plugin, estuviera o no pedido el hook
	fs, err := files(o)
	if err != nil {
		return p, err
	}
	for _, f := range fs {
		old, exists, err := read(f.path)
		if err != nil {
			return p, err
		}
		if exists && core.RemovableFile(old) {
			p.Actions = append(p.Actions, core.Action{Path: f.path, Status: core.Delete, Existed: true, Old: old})
		}
	}
	return p, nil
}

func isAgentHookFile(o Options, path string) bool {
	for name := range settings.Specs {
		if path == adapters.HookConfigPath(o.Home, name) {
			return true
		}
	}
	return false
}
