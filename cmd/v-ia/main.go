package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	corefs "v-ia/core"
	"v-ia/internal/adapters"
	"v-ia/internal/core"
	"v-ia/internal/doctor"
	"v-ia/internal/extras"
	"v-ia/internal/guard"
	"v-ia/internal/hook"
	"v-ia/internal/install"
	"v-ia/internal/lifecycle"
	"v-ia/internal/registry"
	"v-ia/internal/review"
	"v-ia/internal/standards"
	"v-ia/internal/task"
	"v-ia/internal/telemetry"
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
	case "help", "--help", "-h":
		fmt.Print(usageText)
		return 0
	case "version", "--version", "-v":
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
	case "tdd":
		return cmdTDD(args[1:])
	case "telemetry":
		return cmdTelemetry(args[1:])
	case "bench":
		return cmdBench(args[1:])
	case "review":
		return cmdReview(args[1:])
	case "update":
		return cmdUpdate(args[1:])
	case "deps":
		return cmdDeps(args[1:])
	case "standards":
		return cmdStandards(args[1:])
	case "skill-registry":
		return cmdRegistry(args[1:])
	}
	fmt.Fprintf(os.Stderr, "✗ comando desconocido %q (v-ia --help lista los comandos)\n", args[0])
	return 2
}

const usageText = `v-ia ` + adapters.Version + ` — ODD universal para agentes de IA

USO
  v-ia                    TUI interactiva
  v-ia <comando> [flags]

COMANDOS
  install | sync | uninstall   Configura (o retira) ODD en los agentes   [--dry-run --yes --agent --scope global|workspace --preset --persona ...]
  update [--check --yes --channel stable|beta]   Actualiza el binario desde la release (sha256, atestación, minisign)
  upgrade --src DIR            Recompila desde el código fuente
  restore --list | latest --yes | --pin|--unpin [dir]   Backups
  doctor                       Diagnóstico de solo lectura
  mode | tdd                   Modo del proyecto / TDD estricto
  task new|check|verify|status Fichas ODD
  review check|verify|assess|ack|mode   Revisión RDD con recibo
  guard start|check            Huella del árbol
  standards init|check|install|off|status   Estándares del proyecto
  skill-registry list|refresh  Índice de skills
  deps [--yes --only a,b]      Engram, Context7, CodeGraph
  telemetry | bench            Recuento local / coste del hook
  version                      Muestra la versión

Detalle: docs/manual.md. Códigos de salida: 0 bien, 1 fallo, 2 uso incorrecto.
`

func backupRoot(home string) string { return filepath.Join(home, ".v-ia", "backups") }

// cmdPlan sirve a install, sync y uninstall: planifica, muestra el diff y aplica con backup.
func cmdPlan(kind string, args []string) int {
	fs := flag.NewFlagSet(kind, flag.ContinueOnError)
	dry := fs.Bool("dry-run", false, "mostrar diff sin escribir")
	yes := fs.Bool("yes", false, "aplicar sin confirmación")
	force := fs.Bool("force", false, "sobrescribir lo editado a mano (con backup)")
	agent := fs.String("agent", strings.Join(adapters.Primary, ","), "agentes separados por coma ("+agentNames()+", generic) o auto (los detectados en PATH, más claude-code)")
	noHooks := fs.Bool("no-hooks", false, "no instalar los hooks que hacen cumplir ODD")
	noPerms := fs.Bool("no-permissions", false, "no sembrar la deny-list de archivos sensibles")
	engram := fs.Bool("engram", false, "además, configurar Engram (memoria) con 'engram setup' en los agentes elegidos")
	ctx7 := fs.Bool("context7", false, "además, registrar Context7 (documentación por MCP)")
	cgraph := fs.Bool("codegraph", false, "instalar y registrar CodeGraph (grafo de código por MCP); incluido en el preset standard, --codegraph=false lo omite")
	all := fs.Bool("all", false, "además, todos los opcionales: engram, context7 y codegraph")
	persona := fs.String("persona", "neutral", "tono del agente: "+strings.Join(adapters.Personas, " | "))
	preset := fs.String("preset", "standard", "minimal (solo instrucciones) | standard (hooks, skills, deny-list) | full (standard + engram, context7 y codegraph)")
	scope := fs.String("scope", "global", "global (carpeta personal) | workspace (el proyecto actual; solo "+strings.Join(adapters.WorkspaceAgents, ", ")+")")
	if fs.Parse(args) != nil {
		return 2
	}
	noSkills := false
	switch *preset {
	case "standard":
	case "minimal":
		*noHooks, *noPerms, noSkills = true, true, true
	case "full":
		*all = true
	default:
		return fail("preset desconocido: %s (minimal, standard, full)", *preset)
	}
	home, err := adapters.Home()
	if err != nil {
		return fail("%v", err)
	}
	agents := splitList(*agent)
	explicit := false
	fs.Visit(func(f *flag.Flag) { explicit = explicit || f.Name == "agent" })
	if !explicit && kind != "install" {
		if r, ok := install.Load(home); ok && len(r.Agents) > 0 {
			agents = r.Agents
		}
	}
	if len(agents) == 1 && agents[0] == "auto" {
		agents = detectedAgents()
	}
	base := home // dónde se escriben instrucciones, skills y hooks; backups y registro van siempre a home
	switch *scope {
	case "global":
	case "workspace":
		wd, err := os.Getwd()
		if err != nil {
			return fail("%v", err)
		}
		base = wd
		for _, a := range agents {
			if !adapters.SupportsWorkspace(a) {
				return fail("--scope workspace no admite %q (admiten: %s)", a, strings.Join(adapters.WorkspaceAgents, ", "))
			}
		}
	default:
		return fail("scope desconocido: %s (global, workspace)", *scope)
	}
	o := lifecycle.Options{Home: base, Agents: agents, Force: *force, Hooks: !*noHooks, Permissions: !*noPerms, NoSkills: noSkills, Persona: *persona, Engram: *engram || *all}
	for _, a := range o.Agents {
		if _, ok := adapters.Find(a); !ok {
			return fail("agente no soportado: %s (disponibles: %s, generic)", a, agentNames())
		}
	}
	if _, ok := adapters.PersonaBlock(*persona); !ok && *persona != "neutral" && *persona != "" {
		return fail("persona desconocida: %s (%s)", *persona, strings.Join(adapters.Personas, ", "))
	}
	var wanted []string
	for id, on := range map[string]bool{extras.Engram: *engram, extras.Context7: *ctx7, extras.CodeGraph: *cgraph} {
		if on || *all {
			wanted = append(wanted, id)
		}
	}
	if len(wanted) > 0 && kind != "install" {
		return fail("--engram, --context7, --codegraph y --all solo se usan con 'install'")
	}
	var extraSteps []extras.Step
	// CodeGraph forma parte del stack estándar; si faltan requisitos se avisa y se sigue.
	cgraphSet := false
	fs.Visit(func(f *flag.Flag) { cgraphSet = cgraphSet || f.Name == "codegraph" })
	var stackSteps []extras.Step
	if kind == "install" && *preset == "standard" && *scope == "global" && !cgraphSet && !*all {
		steps, notes, err := extras.PlanFor(home, []string{extras.CodeGraph}, o.Agents)
		if err != nil {
			fmt.Println("! aviso: CodeGraph omitido:", err)
		} else {
			stackSteps = steps
			for _, n := range notes {
				fmt.Println("i", n)
			}
		}
	}
	if len(wanted) > 0 {
		sort.Slice(wanted, func(i, j int) bool { return idx(wanted[i]) < idx(wanted[j]) })
		var notes []string
		extraSteps, notes, err = extras.PlanFor(home, wanted, o.Agents)
		if err != nil {
			if !(extras.IsMissing(err) && *dry) {
				return fail("%v", err)
			}
			fmt.Println("!", err, "(dry-run: se muestra lo que haría; sin ellos --yes se negaría a aplicar)")
		}
		defer func() {
			for _, n := range notes {
				fmt.Println("i", n)
			}
		}()
	}
	extraSteps = append(extraSteps, stackSteps...)
	plan := func() (core.Plan, error) {
		switch kind {
		case "install":
			return lifecycle.Install(o)
		case "sync":
			return lifecycle.Sync(o)
		}
		return lifecycle.Uninstall(o)
	}
	p, err := plan()
	if err != nil {
		return fail("%v", err)
	}
	for _, w := range p.Warnings {
		fmt.Println("! aviso:", w)
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
	for _, st := range extraSteps {
		fmt.Printf("$ %s   # %s\n", st, st.Desc)
	}
	if *dry {
		fmt.Println("dry-run: no se escribió ni ejecutó nada")
		return 0
	}
	if !*yes && (len(p.Changes()) > 0 || len(extraSteps) > 0) {
		return fail("falta --yes para aplicar (o usa --dry-run para previsualizar)")
	}
	// Los componentes externos van primero: si uno falla no se ha escrito nada de V-IA.
	// Editan archivos que V-IA no controla, así que antes se respaldan y después se vuelve
	// a planificar, para no pisar lo que añadan.
	var before core.Snapshot
	if len(extraSteps) > 0 {
		paths := extras.ConfigFiles(home)
		for _, a := range p.Changes() {
			paths = append(paths, a.Path)
		}
		if before, err = core.Capture(paths); err != nil {
			return fail("no pude respaldar antes de ejecutar instaladores externos: %v", err)
		}
		if err := extras.Run(extraSteps, os.Stdout); err != nil {
			saved, serr := before.Save(backupRoot(home), time.Now())
			if serr != nil {
				return fail("%v; además falló el respaldo: %v", err, serr)
			}
			return fail("%v (no se escribió ningún archivo de V-IA; estado anterior en %s, 'v-ia restore latest --yes' lo recupera)", err, saved)
		}
		if p, err = plan(); err != nil {
			return fail("%v", err)
		}
	}
	dir, err := core.Apply(p, backupRoot(home), time.Now())
	if err != nil {
		return fail("%v", err)
	}
	if before != nil { // queda como backup más reciente: deshace todo, no solo lo de V-IA
		if dir, err = before.Save(backupRoot(home), time.Now()); err != nil {
			return fail("%v", err)
		}
	}
	switch {
	case *scope == "workspace": // el registro de instalación es global: no se toca
	case kind == "install":
		_ = install.Add(home, install.Record{Version: adapters.Version, Preset: *preset, Agents: o.Agents, Components: wanted, Hooks: o.Hooks, Skills: !o.NoSkills, Updated: time.Now().UTC()})
	case kind == "uninstall":
		_ = install.Remove(home, o.Agents)
	}
	if dir == "" {
		fmt.Println("✓ nada que cambiar")
	} else {
		fmt.Println("✓ aplicado; backup en", dir)
	}
	if len(extraSteps) > 0 {
		fmt.Println("✓ componentes opcionales listos; reinicia tu agente para cargar los servidores MCP")
	}
	return 0
}

func idx(id string) int {
	for i, x := range extras.All {
		if x == id {
			return i
		}
	}
	return len(extras.All)
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
		return fail("uso: restore --list | restore latest --yes | restore --pin|--unpin [dir]")
	}
	switch args[0] {
	case "--list":
		l, err := core.ListBackups(root)
		if err != nil {
			return fail("%v", err)
		}
		for _, b := range l {
			if core.IsPinned(b) {
				fmt.Println(b, "[pinned]")
				continue
			}
			fmt.Println(b)
		}
		return 0
	case "--pin", "--unpin":
		d, err := core.Latest(root)
		if len(args) > 1 {
			d, err = args[1], nil
		}
		if err != nil {
			return fail("%v", err)
		}
		if args[0] == "--pin" {
			err = core.Pin(d)
		} else {
			err = core.Unpin(d)
		}
		if err != nil {
			return fail("%v", err)
		}
		fmt.Println("✓", args[0][2:], d)
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
	return fail("uso: restore --list | restore latest --yes | restore --pin|--unpin [dir]")
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
		return fail("uso: task new <feature> [--request \"texto literal\"] | task check | task verify [--strict] | task status")
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
	case "verify":
		return cmdTaskVerify(root, len(args) > 1 && args[1] == "--strict")
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
	return fail("uso: task new <feature> | task check | task verify [--strict] | task status")
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
	cursor := in.Event == "preToolUse" || in.Event == "beforeSubmitPrompt" // Cursor exige JSON en stdout
	r := hook.Handle(in)
	if r.Block {
		if h, err := adapters.Home(); err == nil {
			telemetry.Record(h, "block:"+hook.Category(r.Msg))
		}
		fmt.Fprintln(os.Stderr, r.Msg)
		return 2
	}
	if cursor {
		if in.Event == "beforeSubmitPrompt" {
			fmt.Println(`{"continue":true}`)
		} else {
			fmt.Println(`{"permission":"allow"}`)
		}
		return 0
	}
	if r.Msg != "" && in.TurnID != "" { // Codex: el texto plano cuenta como hook fallido; espera JSON
		b, _ := json.Marshal(map[string]any{"hookSpecificOutput": map[string]any{"hookEventName": in.Event, "additionalContext": r.Msg}})
		fmt.Println(string(b))
		return 0
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
		return fail("uso: review check | status | verify | assess | ack | refute | inspect | reset | mode status|enable|disable")
	}
	switch args[0] {
	case "check":
		cfs := flag.NewFlagSet("check", flag.ContinueOnError)
		lenses := cfs.String("lenses", "", "lentes a revisar (risk,resilience,readability,reliability); solo en medium/high; con la revisión abierta se pueden añadir, no quitar")
		lreason := cfs.String("lenses-reason", "", "razón de la selección de lentes (obligatoria con --lenses)")
		escItem := cfs.Int("escalate-item", 0, "sube a riesgo ALTO por un ítem cerrado 1-6 (datos, seguridad, contratos, concurrencia, entrega, sin test)")
		escReason := cfs.String("escalate-reason", "", "razón del escalado (una línea, ≤500 bytes)")
		baseRef := cfs.String("base-ref", "", "referencia base (por defecto HEAD)")
		request := cfs.String("request", "", "petición literal del usuario; queda congelada en el recibo")
		requestFile := cfs.String("request-file", "", "archivo con la petición literal (p. ej. la ficha ODD)")
		if cfs.Parse(args[1:]) != nil {
			return 2
		}
		if *requestFile != "" {
			if *request != "" {
				return fail("usa --request o --request-file, no ambos")
			}
			b, err := os.ReadFile(*requestFile)
			if err != nil {
				return fail("%v", err)
			}
			*request = string(b)
		}
		r, err := review.Run(root, review.DetectCommands(root), review.RunOptions{Request: *request, Base: *baseRef, Lenses: splitList(*lenses), LensReason: *lreason, EscItem: *escItem, EscReason: *escReason})
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
		if rules, found, err := standards.Load(root); found && err == nil {
			if files, err := standards.Changes(root, standards.Head); err == nil {
				for _, f := range standards.Deterministic(rules, files) {
					fmt.Println("! estándares:", f)
				}
			}
		}
		fmt.Printf("riesgo: %s (%s)\n", r.Risk.Tier, strings.Join(r.Risk.Reasons, "; "))
		fmt.Printf("recibo: %s (candidato %.12s, intento %d)\n", r.Result, r.CandidateID, r.Attempt)
		if on, _ := review.Mode(root, home); on && r.Risk.ReviewDue && r.Result == "pass" {
			fmt.Printf("! revisión independiente debida (%s): lanza v-ia-reviewer por cada lente (%s) y registra con 'v-ia review ack --input FILE'\n", r.Risk.ReviewDueReason, strings.Join(r.SelectedLenses, ", "))
			fmt.Println("  rutas a cubrir:", strings.Join(r.Paths, " "))
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
		afs := flag.NewFlagSet("ack", flag.ContinueOnError)
		input := afs.String("input", "-", "JSON de la revisión de una lente (- = stdin)")
		if afs.Parse(args[1:]) != nil {
			return 2
		}
		var raw []byte
		var rerr error
		if *input == "-" {
			raw, rerr = io.ReadAll(os.Stdin)
		} else {
			raw, rerr = os.ReadFile(*input)
		}
		if rerr != nil {
			return fail("%v", rerr)
		}
		var in review.ReviewInput
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields() // esquema estricto: un campo desconocido se rechaza antes de cambiar nada
		if err := dec.Decode(&in); err != nil {
			return fail("revisión inválida: %v", err)
		}
		r, err := review.RecordReview(root, in)
		if err != nil {
			return fail("%v", err)
		}
		for _, x := range r.Reviews {
			if x.Lens == in.Lens {
				fmt.Printf("✓ lente %s admitida: veredicto %s (calculado), %d hallazgo(s)\n", x.Lens, x.Verdict, len(x.Findings))
			}
		}
		st := review.Status(root)
		fmt.Printf("estado: %s (%s)", st.State, st.Detail)
		if len(st.Pending) > 0 {
			fmt.Printf("; faltan lentes: %s", strings.Join(st.Pending, ", "))
		}
		fmt.Println()
		return 0
	case "refute":
		rfs := flag.NewFlagSet("refute", flag.ContinueOnError)
		input := rfs.String("input", "-", "JSON de la refutación (- = stdin)")
		if rfs.Parse(args[1:]) != nil {
			return 2
		}
		var raw []byte
		var rerr error
		if *input == "-" {
			raw, rerr = io.ReadAll(os.Stdin)
		} else {
			raw, rerr = os.ReadFile(*input)
		}
		if rerr != nil {
			return fail("%v", rerr)
		}
		var in review.RefuteInput
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil {
			return fail("refutación inválida: %v", err)
		}
		r, err := review.Refute(root, in)
		if err != nil {
			return fail("%v", err)
		}
		fmt.Printf("✓ refutación admitida: estado %s, %d bloqueante(s) en pie\n", r.State, len(r.Blocking()))
		return 0
	case "status":
		st := review.Status(root)
		if len(args) > 1 && args[1] == "--json" {
			b, _ := json.MarshalIndent(st, "", "  ")
			fmt.Println(string(b))
			return 0
		}
		fmt.Printf("%s (%s) → %s\n", st.State, st.Detail, st.NextTransition)
		return 0
	case "inspect":
		ifs := flag.NewFlagSet("inspect", flag.ContinueOnError)
		op := ifs.String("op", "name-status", "name-status | numstat | stat | patch | object")
		path := ifs.String("path", "", "ruta (patch u object)")
		if ifs.Parse(args[1:]) != nil {
			return 2
		}
		r, err := review.Load(root)
		if err != nil {
			return fail("%v", err)
		}
		out, err := review.Inspect(root, r, *op, *path)
		if err != nil {
			return fail("%v", err)
		}
		fmt.Print(out)
		return 0
	case "reset":
		if err := review.Reset(root); err != nil {
			return fail("%v", err)
		}
		fmt.Println("✓ recibo descartado: la siguiente 'review check' empieza una revisión nueva")
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

// cmdTDD activa o desactiva el TDD estricto del proyecto actual.
func cmdTDD(args []string) int {
	root, _ := os.Getwd()
	if len(args) == 0 || args[0] == "status" {
		if hook.TDDEnabled(root) {
			fmt.Println("tdd estricto: on (no se edita código de producción sin una prueba nueva o modificada)")
		} else {
			fmt.Println("tdd estricto: off")
		}
		return 0
	}
	switch args[0] {
	case "on", "off":
		if err := hook.SetTDD(root, args[0] == "on"); err != nil {
			return fail("%v", err)
		}
		fmt.Println("✓ tdd estricto:", args[0])
		return 0
	}
	return fail("uso: tdd on | off | status")
}

// cmdTaskVerify contrasta la evidencia de las fichas con el recibo vigente de `review check`.
func cmdTaskVerify(root string, strict bool) int {
	if err := review.Verify(root, false); err != nil {
		return fail("no hay recibo vigente con el que contrastar: %v", err)
	}
	r, err := review.Load(root)
	if err != nil {
		return fail("%v", err)
	}
	var ran []string
	for _, c := range r.Checks {
		if c.Exit == 0 {
			ran = append(ran, c.Cmd)
		}
	}
	files, _ := filepath.Glob(filepath.Join(root, "odd", "tasks", "*.md"))
	bad := 0
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return fail("%v", err)
		}
		backed, unbacked := task.Verify(string(b), ran)
		fmt.Printf("%s: %d respaldadas por el recibo, %d sin respaldo\n", filepath.Base(f), len(backed), len(unbacked))
		for _, id := range unbacked {
			fmt.Printf("  ! %s: su evidencia no cita ninguna comprobación ejecutada por 'review check'\n", id)
		}
		bad += len(unbacked)
	}
	if bad > 0 && strict {
		return 1
	}
	return 0
}

// detectedAgents devuelve claude-code (el principal) y los demás agentes que haya en PATH.
func detectedAgents() []string {
	out := []string{"claude-code"}
	for _, a := range adapters.All {
		if a.Name != "claude-code" && a.Detected() {
			out = append(out, a.Name)
		}
	}
	return out
}

// cmdTelemetry gestiona el recuento local (sin red) de bloqueos.
func cmdTelemetry(args []string) int {
	home, err := adapters.Home()
	if err != nil {
		return fail("%v", err)
	}
	if len(args) == 0 || args[0] == "status" {
		state := "off"
		if telemetry.Enabled(home) {
			state = "on"
		}
		fmt.Println("recuento local:", state, "(solo en este equipo; no hay envío por red)")
		return 0
	}
	switch args[0] {
	case "on", "off":
		if err := telemetry.Set(home, args[0] == "on"); err != nil {
			return fail("%v", err)
		}
		fmt.Println("✓ recuento local:", args[0])
	case "show":
		d := telemetry.Load(home)
		if len(d.Counts) == 0 {
			fmt.Println("sin datos (¿activaste `v-ia telemetry on`?)")
		}
		for _, e := range d.Sorted() {
			fmt.Printf("%-24s %d\n", e, d.Counts[e])
		}
	case "reset":
		if err := telemetry.Reset(home); err != nil {
			return fail("%v", err)
		}
		fmt.Println("✓ contadores borrados")
	default:
		return fail("uso: telemetry on | off | status | show | reset")
	}
	return 0
}

// cmdBench mide cuánto tarda V-IA en decidir cada tipo de acción (lo que añade a cada herramienta).
func cmdBench(args []string) int {
	n := 200
	if len(args) > 1 && args[0] == "-n" {
		if v, err := strconv.Atoi(args[1]); err == nil && v > 0 {
			n = v
		}
	}
	res, err := hook.Bench(n)
	if err != nil {
		return fail("%v", err)
	}
	fmt.Printf("%-34s %10s %10s %10s\n", "decisión del hook ("+strconv.Itoa(n)+" iteraciones)", "mediana", "p95", "máx")
	for _, r := range res {
		fmt.Printf("%-34s %10s %10s %10s\n", r.Name, r.P50, r.P95, r.Max)
	}
	return 0
}
