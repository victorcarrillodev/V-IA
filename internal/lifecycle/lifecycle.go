// Package lifecycle construye los planes de install, sync y uninstall.
// Solo planifica: aplicar (backup, escritura atómica, rollback) es de core.Apply.
package lifecycle

import (
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
}

var skillNames = []string{"odd-resume", "odd-verify", "odd-delegate", "rdd-review", "work-unit-commits", "branch-pr", "issue-creation", "chained-pr", "judgment-day", "skill-creator", "comment-writer", "rdd-defect-workflow"}

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

type managedFile struct{ path, content string }

// files son los archivos completos que V-IA gestiona para Claude Code.
func files(home string) ([]managedFile, error) {
	var out []managedFile
	for _, n := range skillNames {
		b, err := corefs.FS.ReadFile("skills/" + n + "/SKILL.md")
		if err != nil {
			return nil, err
		}
		out = append(out, managedFile{filepath.Join(home, ".claude", "skills", n, "SKILL.md"), string(b)})
	}
	b, err := corefs.FS.ReadFile("agents/v-ia-reviewer.md")
	if err != nil {
		return nil, err
	}
	return append(out, managedFile{filepath.Join(home, ".claude", "agents", "v-ia-reviewer.md"), string(b)}), nil
}

func fileAction(f managedFile, force bool) (core.Action, error) {
	old, exists, err := read(f.path)
	if err != nil {
		return core.Action{}, err
	}
	nw, st := core.UpsertFile(old, exists, f.content, force)
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

// Install planifica bloques de instrucciones, y para Claude Code además hooks, deny-list,
// skills y el subagente revisor.
func Install(o Options) (core.Plan, error) {
	if o.Exe == "" {
		o.Exe = ExeForHooks()
	}
	ts, err := adapters.Targets(o.Home, o.Agents)
	if err != nil {
		return core.Plan{}, err
	}
	p, err := core.BuildPlan(ts, o.Force)
	if err != nil {
		return core.Plan{}, err
	}
	if has(o.Agents, "claude-code") {
		if o.Hooks || o.Permissions {
			a, err := settingsAction(o)
			if err != nil {
				return core.Plan{}, err
			}
			p.Actions = append(p.Actions, a)
		}
		fs, err := files(o.Home)
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
		}
		p.Actions = append(p.Actions, a)
	}
	return p, nil
}

// Uninstall retira bloques, hooks y archivos gestionados intactos. Lo editado a mano se
// conserva. La deny-list de permisos es decisión del usuario y se deja.
func Uninstall(o Options) (core.Plan, error) {
	var p core.Plan
	ts, err := adapters.Targets(o.Home, o.Agents)
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
		nw, removed := core.Remove(old, t.Block.ID)
		switch {
		case !removed:
			p.Actions = append(p.Actions, core.Action{Path: t.Path, Status: core.Unchanged, Existed: true, Old: old, New: old})
		case strings.TrimSpace(nw) == "":
			p.Actions = append(p.Actions, core.Action{Path: t.Path, Status: core.Delete, Existed: true, Old: old})
		default:
			p.Actions = append(p.Actions, core.Action{Path: t.Path, Status: core.Update, Existed: true, Old: old, New: nw})
		}
	}
	if !has(o.Agents, "claude-code") {
		return p, nil
	}
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
	fs, err := files(o.Home)
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
