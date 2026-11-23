package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	corefs "v-ia/core"
	"v-ia/internal/adapters"
	"v-ia/internal/core"
	"v-ia/internal/doctor"
	"v-ia/internal/guard"
	"v-ia/internal/hook"
	"v-ia/internal/lifecycle"
	"v-ia/internal/registry"
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
		return cmdPlan("install", args[1:])
	case "sync":
		return cmdPlan("sync", args[1:])
	case "uninstall":
		return cmdPlan("uninstall", args[1:])
	case "upgrade":
		return cmdUpgrade(args[1:])
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
	case "skill-registry":
		return cmdRegistry(args[1:])
	}
	return fail("comando desconocido %q", args[0])
}

func backupRoot(home string) string { return filepath.Join(home, ".v-ia", "backups") }

// cmdPlan sirve a install, sync y uninstall: planifica, muestra el diff y aplica con backup.
func cmdPlan(kind string, args []string) int {
	fs := flag.NewFlagSet(kind, flag.ContinueOnError)
	dry := fs.Bool("dry-run", false, "mostrar diff sin escribir")
	yes := fs.Bool("yes", false, "aplicar sin confirmación")
	force := fs.Bool("force", false, "sobrescribir lo editado a mano (con backup)")
	agent := fs.String("agent", "claude-code", "agentes separados por coma ("+agentNames()+", generic)")
	noHooks := fs.Bool("no-hooks", false, "no instalar los hooks que hacen cumplir ODD")
	noPerms := fs.Bool("no-permissions", false, "no sembrar la deny-list de archivos sensibles")
	engram := fs.Bool("engram", false, "tras instalar, ejecutar 'engram setup claude-code'")
	if fs.Parse(args) != nil {
		return 2
	}
	home, err := adapters.Home()
	if err != nil {
		return fail("%v", err)
	}
	o := lifecycle.Options{Home: home, Agents: splitList(*agent), Force: *force, Hooks: !*noHooks, Permissions: !*noPerms}
	for _, a := range o.Agents {
		if _, ok := adapters.Find(a); !ok {
			return fail("agente no soportado: %s (disponibles: %s, generic)", a, agentNames())
		}
	}
	var p core.Plan
	switch kind {
	case "install":
		p, err = lifecycle.Install(o)
	case "sync":
		p, err = lifecycle.Sync(o)
	default:
		p, err = lifecycle.Uninstall(o)
	}
	if err != nil {
		return fail("%v", err)
	}
	for _, a := range p.Actions {
		switch a.Status {
		case core.Unchanged:
			fmt.Printf("= %s (sin cambios)\n", a.Path)
		case core.Delete:
			fmt.Printf("--- %s (delete)\n", a.Path)
		case core.Conflict:
			fmt.Printf("! %s: editado a mano; usa --force para sobrescribir (con backup)\n", a.Path)
		default:
			fmt.Print(a.Diff())
		}
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
	if kind == "install" && *engram {
		return setupEngram()
	}
	return 0
}

func setupEngram() int {
	exe, err := exec.LookPath("engram")
	if err != nil {
		return fail("engram no está en PATH: instálalo (https://github.com/Gentleman-Programming/engram) y repite con --engram")
	}
	cmd := exec.Command(exe, "setup", "claude-code")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fail("engram setup claude-code: %v", err)
	}
	fmt.Println("✓ engram configurado para claude-code")
	return 0
}

func agentNames() string {
	var n []string
	for _, a := range adapters.All {
		n = append(n, a.Name)
	}
	return strings.Join(n, ", ")
}

func splitList(s string) []string {
	var out []string
	for _, x := range strings.Split(s, ",") {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}

// cmdUpgrade recompila desde el código fuente y reemplaza el binario en uso.
func cmdUpgrade(args []string) int {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	src := fs.String("src", os.Getenv("V_IA_SRC"), "directorio del repositorio de V-IA (o V_IA_SRC)")
	if fs.Parse(args) != nil {
		return 2
	}
	if *src == "" {
		return fail("indica el repositorio: v-ia upgrade --src <dir> (o V_IA_SRC)")
	}
	self, err := os.Executable()
	if err != nil {
		return fail("%v", err)
	}
	if out, err := exec.Command("git", "-C", *src, "pull", "--ff-only").CombinedOutput(); err != nil {
		return fail("git pull: %v\n%s", err, out)
	}
	tmp := self + ".new"
	build := exec.Command("go", "build", "-o", tmp, "./cmd/v-ia")
	build.Dir = *src
	if out, err := build.CombinedOutput(); err != nil {
		os.Remove(tmp)
		return fail("go build: %v\n%s", err, out)
	}
	if err := os.Rename(tmp, self); err != nil {
		os.Remove(tmp)
		return fail("reemplazando %s: %v", self, err)
	}
	fmt.Println("✓ actualizado; ejecuta 'v-ia sync' para refrescar los archivos gestionados")
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
		return fail("sin terminal interactiva: usa flags (install | sync | uninstall | doctor | restore | task | guard | review | version)")
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
		return fail("uso: task new <feature> [--request \"texto literal\"] | task check | task status")
	}
	root, _ := os.Getwd()
	switch args[0] {
	case "new":
		rest, req := args[1:], ""
		for i, a := range rest {
			if a == "--request" && i+1 < len(rest) {
				req, rest = strings.Join(rest[i+1:], " "), rest[:i]
				break
			}
		}
		if len(rest) < 1 {
			return fail("uso: task new <feature> [--request \"texto literal\"]")
		}
		tmpl, err := corefs.FS.ReadFile("odd/templates/task.md")
		if err != nil {
			return fail("%v", err)
		}
		p, err := task.New(root, strings.Join(rest, " "), string(tmpl), time.Now(), req)
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
	case "status":
		out, err := task.Status(root)
		if err != nil {
			return fail("%v", err)
		}
		if len(out) == 0 {
			fmt.Println("sin fichas en odd/tasks/")
		}
		for _, l := range out {
			fmt.Println(l)
		}
		return 0
	}
	return fail("uso: task new <feature> | task check | task status")
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

// cmdHook lo invocan los hooks de Claude Code. Falla abierto: ante un error propio
// (JSON ilegible, etc.) no bloquea al agente; solo bloquea cuando una regla se incumple.
func cmdHook(args []string) int {
	var in hook.Input
	if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
		return 0
	}
	if in.Event == "" && len(args) > 0 {
		in.Event = map[string]string{"pre": "PreToolUse", "prompt": "UserPromptSubmit", "stop": "Stop", "session": "SessionStart"}[args[0]]
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
	home, err := adapters.Home()
	if err != nil {
		return fail("%v", err)
	}
	if len(args) == 0 {
		return fail("uso: review check | verify | assess | ack | mode status|enable|disable")
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
		fmt.Printf("riesgo: %s (%s)\n", r.Risk.Tier, strings.Join(r.Risk.Reasons, "; "))
		fmt.Printf("recibo: %s (candidato %.12s)\n", r.Result, r.CandidateID)
		if on, _ := review.Mode(root, home); on && r.Risk.ReviewDue {
			fmt.Println("! riesgo ALTO: antes de entregar, lanza el subagente v-ia-reviewer (revisa y registra con 'v-ia review ack')")
		}
		if r.Result != "pass" {
			return 1
		}
		return 0
	case "verify":
		on, _ := review.Mode(root, home)
		if err := review.Verify(root, on); err != nil {
			return fail("%v", err)
		}
		fmt.Println("✓ recibo válido para el árbol actual")
		return 0
	case "assess":
		fs := flag.NewFlagSet("assess", flag.ContinueOnError)
		base := fs.String("base-ref", "HEAD", "referencia base")
		if fs.Parse(args[1:]) != nil {
			return 2
		}
		a := review.Assess(root, *base)
		b, _ := json.MarshalIndent(a, "", "  ")
		fmt.Println(string(b))
		return 0
	case "ack":
		fs := flag.NewFlagSet("ack", flag.ContinueOnError)
		verdict := fs.String("verdict", "", "approve | changes")
		reviewer := fs.String("reviewer", "manual", "quién revisó")
		note := fs.String("note", "", "resumen")
		if fs.Parse(args[1:]) != nil {
			return 2
		}
		if err := review.RecordAck(root, *verdict, *reviewer, *note); err != nil {
			return fail("%v", err)
		}
		fmt.Println("✓ veredicto registrado:", *verdict)
		return 0
	case "mode":
		sub := "status"
		if len(args) > 1 {
			sub = args[1]
		}
		fs := flag.NewFlagSet("mode", flag.ContinueOnError)
		scope := fs.String("scope", "project", "global | project")
		if len(args) > 2 && fs.Parse(args[2:]) != nil {
			return 2
		}
		switch sub {
		case "status":
			on, src := review.Mode(root, home)
			fmt.Printf("RDD: %v (fuente: %s)\n", map[bool]string{true: "ON", false: "OFF"}[on], src)
			return 0
		case "enable", "disable":
			if err := review.SetMode(root, home, *scope, sub == "enable"); err != nil {
				return fail("%v", err)
			}
			fmt.Printf("✓ RDD %sd (%s)\n", sub, *scope)
			return 0
		}
	}
	return fail("uso: review check | verify | assess | ack | mode status|enable|disable [--scope global|project]")
}

func cmdRegistry(args []string) int {
	root, _ := os.Getwd()
	home, err := adapters.Home()
	if err != nil {
		return fail("%v", err)
	}
	if len(args) == 0 {
		return fail("uso: skill-registry list | refresh")
	}
	switch args[0] {
	case "list":
		for _, s := range registry.Scan(home, root) {
			fmt.Printf("%s\t%s\t%s\n", s.Name, s.Scope, s.Description)
		}
		return 0
	case "refresh":
		w, err := registry.Refresh(home, root)
		if err != nil {
			return fail("%v", err)
		}
		if w {
			fmt.Println("✓ registro actualizado:", registry.Path(root))
		} else {
			fmt.Println("✓ registro sin cambios")
		}
		return 0
	}
	return fail("uso: skill-registry list | refresh")
}
