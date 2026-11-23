package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	corefs "v-ia/core"
	"v-ia/internal/adapters"
	"v-ia/internal/core"
	"v-ia/internal/doctor"
	"v-ia/internal/guard"
	"v-ia/internal/hook"
	"v-ia/internal/review"
	"v-ia/internal/task"
	"v-ia/internal/tui"
)

func main() { os.Exit(run(os.Args[1:])) }

func fail(format string, a ...any) int {
	fmt.Fprintf(os.Stderr, "✗ "+format+"\n", a...)
	return 1
}

func run(args []string) int {
	if len(args) == 0 {
		return cmdTUI()
	}
	switch args[0] {
	case "version":
		fmt.Println("v-ia", adapters.Version)
		return 0
	case "install":
		return cmdInstall(args[1:])
	case "restore":
		return cmdRestore(args[1:])
	case "doctor":
		return cmdDoctor()
	case "task":
		return cmdTask(args[1:])
	case "guard":
		return cmdGuard(args[1:])
	case "hook":
		return cmdHook(args[1:])
	case "mode":
		return cmdMode(args[1:])
	case "review":
		return cmdReview(args[1:])
	}
	return fail("comando desconocido %q", args[0])
}

func backupRoot(home string) string { return filepath.Join(home, ".v-ia", "backups") }

func cmdInstall(args []string) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	dry := fs.Bool("dry-run", false, "mostrar diff sin escribir")
	yes := fs.Bool("yes", false, "aplicar sin confirmación")
	force := fs.Bool("force", false, "sobrescribir bloque editado a mano (con backup)")
	agent := fs.String("agent", "claude-code", "agentes separados por coma (claude-code, generic)")
	noHooks := fs.Bool("no-hooks", false, "no instalar los hooks que hacen cumplir ODD")
	if fs.Parse(args) != nil {
		return 2
	}
	home, err := adapters.Home()
	if err != nil {
		return fail("%v", err)
	}
	ts, err := adapters.Targets(home, strings.Split(*agent, ","))
	if err != nil || len(ts) == 0 {
		return fail("agente no soportado: %s (%v)", *agent, err)
	}
	p, err := core.BuildPlan(ts, *force)
	if err != nil {
		return fail("%v", err)
	}
	if !*noHooks && containsStr(strings.Split(*agent, ","), "claude-code") {
		sa, err := adapters.SettingsAction(home)
		if err != nil {
			return fail("hooks: %v", err)
		}
		p.Actions = append(p.Actions, sa)
	}
	for _, a := range p.Actions {
		if a.Status == core.Unchanged {
			fmt.Printf("= %s (sin cambios)\n", a.Path)
			continue
		}
		fmt.Print(a.Diff())
	}
	for _, c := range p.Conflicts() {
		fmt.Printf("! %s: el bloque gestionado fue editado a mano; usa --force para sobrescribir (con backup)\n", c.Path)
	}
	if *dry {
		fmt.Println("dry-run: no se escribió nada")
		return 0
	}
	if !*yes && len(p.Changes()) > 0 {
		return fail("falta --yes para aplicar (o usa --dry-run para previsualizar)")
	}
	dir, err := core.Apply(p, backupRoot(home), time.Now())
	if err != nil {
		return fail("%v", err)
	}
	if dir == "" {
		fmt.Println("✓ nada que cambiar")
	} else {
		fmt.Println("✓ aplicado; backup en", dir)
	}
	return 0
}

func cmdRestore(args []string) int {
	home, err := adapters.Home()
	if err != nil {
		return fail("%v", err)
	}
	root := backupRoot(home)
	if len(args) == 0 {
		return fail("uso: restore --list | restore latest --yes")
	}
	switch args[0] {
	case "--list":
		l, err := core.ListBackups(root)
		if err != nil {
			return fail("%v", err)
		}
		for _, b := range l {
			fmt.Println(b)
		}
		return 0
	case "latest":
		if len(args) < 2 || args[1] != "--yes" {
			return fail("restore latest requiere --yes")
		}
		d, err := core.Latest(root)
		if err != nil {
			return fail("%v", err)
		}
		if err := core.Restore(d); err != nil {
			return fail("%v", err)
		}
		fmt.Println("✓ restaurado desde", d)
		return 0
	}
	return fail("uso: restore --list | restore latest --yes")
}

func cmdDoctor() int {
	home, err := adapters.Home()
	if err != nil {
		return fail("%v", err)
	}
	rs := doctor.Run(home)
	for _, r := range rs {
		fmt.Printf("%s %s\n", r.Level.Symbol(), r.Msg)
	}
	if doctor.Failed(rs) {
		return 1
	}
	return 0
}

func isTTY(f *os.File) bool {
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func cmdTUI() int {
	if !isTTY(os.Stdin) || !isTTY(os.Stdout) {
		return fail("sin terminal interactiva: usa flags (install --dry-run | install --yes | doctor | restore | task | guard | version)")
	}
	home, err := adapters.Home()
	if err != nil {
		return fail("%v", err)
	}
	if err := tui.Run(home); err != nil {
		return fail("%v", err)
	}
	return 0
}

func cmdTask(args []string) int {
	if len(args) == 0 {
		return fail("uso: task new <feature> | task check")
	}
	root, _ := os.Getwd()
	switch args[0] {
	case "new":
		if len(args) < 2 {
			return fail("uso: task new <feature>")
		}
		tmpl, err := corefs.FS.ReadFile("odd/templates/task.md")
		if err != nil {
			return fail("%v", err)
		}
		p, err := task.New(root, strings.Join(args[1:], " "), string(tmpl), time.Now())
		if err != nil {
			return fail("%v", err)
		}
		fmt.Println("✓ creado", p)
		return 0
	case "check":
		res, n, err := task.CheckAll(root)
		if err != nil {
			return fail("%v", err)
		}
		if len(res) == 0 {
			fmt.Printf("✓ %d tarea(s) válidas\n", n)
			return 0
		}
		for f, ps := range res {
			for _, p := range ps {
				fmt.Printf("✗ %s: %s\n", f, p)
			}
		}
		return 1
	}
	return fail("uso: task new <feature> | task check")
}

func cmdGuard(args []string) int {
	root, _ := os.Getwd()
	if len(args) == 0 {
		return fail("uso: guard start | guard check")
	}
	switch args[0] {
	case "start":
		if err := guard.Start(root); err != nil {
			return fail("%v", err)
		}
		fmt.Println("✓ referencia guardada")
		return 0
	case "check":
		ch, err := guard.Diff(root)
		if err != nil {
			return fail("%v (¿falta 'guard start'?)", err)
		}
		if len(ch) == 0 {
			fmt.Println("✓ sin cambios")
			return 0
		}
		for _, c := range ch {
			fmt.Println("✗", c)
		}
		return 1
	}
	return fail("uso: guard start | guard check")
}

func containsStr(l []string, s string) bool {
	for _, x := range l {
		if strings.TrimSpace(x) == s {
			return true
		}
	}
	return false
}

// cmdHook lo invocan los hooks de Claude Code. Falla abierto: ante un error propio
// (JSON ilegible, etc.) no bloquea al agente; solo bloquea cuando una regla ODD se incumple.
func cmdHook(args []string) int {
	var in hook.Input
	if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
		return 0
	}
	if in.Event == "" && len(args) > 0 {
		in.Event = map[string]string{"pre": "PreToolUse", "prompt": "UserPromptSubmit", "stop": "Stop"}[args[0]]
	}
	r := hook.Handle(in)
	if r.Block {
		fmt.Fprintln(os.Stderr, r.Msg)
		return 2
	}
	if r.Msg != "" {
		fmt.Println(r.Msg)
	}
	return 0
}

func cmdMode(args []string) int {
	root, _ := os.Getwd()
	if len(args) == 0 || args[0] == "status" {
		m := hook.LoadState(root).Mode
		if m == "" {
			m = "(sin clasificar)"
		}
		fmt.Println("modo:", m)
		return 0
	}
	switch args[0] {
	case hook.ModeRead, hook.ModeEdit, hook.ModeTracking:
		st := hook.LoadState(root)
		st.Mode = args[0]
		if err := hook.SaveState(root, st); err != nil {
			return fail("%v", err)
		}
		fmt.Println("✓ modo:", args[0])
		return 0
	}
	return fail("uso: mode [status|read|edit|tracking]")
}

func cmdReview(args []string) int {
	root, _ := os.Getwd()
	if len(args) == 0 {
		return fail("uso: review check | review verify")
	}
	switch args[0] {
	case "check":
		r, err := review.Run(root, review.DetectCommands(root), 10*time.Minute)
		if err != nil {
			return fail("%v", err)
		}
		for _, c := range r.Checks {
			mark := "✓"
			if c.Exit != 0 {
				mark = "✗"
			}
			fmt.Printf("%s %s\n", mark, c.Cmd)
			if c.Exit != 0 {
				fmt.Println(c.Output)
			}
		}
		fmt.Printf("recibo: %s (candidato %.12s)\n", r.Result, r.CandidateID)
		if r.Result != "pass" {
			return 1
		}
		return 0
	case "verify":
		if err := review.Verify(root); err != nil {
			return fail("%v", err)
		}
		fmt.Println("✓ recibo válido para el árbol actual")
		return 0
	}
	return fail("uso: review check | review verify")
}
