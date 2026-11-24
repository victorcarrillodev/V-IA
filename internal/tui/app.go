// Package tui es la capa interactiva (Bubble Tea) sobre los paquetes de lógica.
// Solo construye planes y llama a core/lifecycle; no contiene lógica de instalación.
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"v-ia/internal/adapters"
	"v-ia/internal/core"
	"v-ia/internal/doctor"
	"v-ia/internal/lifecycle"
)

type screen int

const (
	scHome screen = iota
	scAgents
	scComponents
	scReview
	scApply
	scDone
	scStatus
	scProject
	scRestore
)

// op es la operación que recorre el flujo Revisión → Aplicar → Final.
type op int

const (
	opInstall op = iota
	opSync
	opUninstall
)

var opDone = [...]string{"Aplicado.", "Sincronizado.", "Desinstalado."}

type menuItem struct{ label, desc string }

var menu = []menuItem{
	{"Instalar / configurar agentes", "bloques ODD, hooks, skills y subagente revisor"},
	{"Estado del sistema", "diagnóstico de solo lectura (doctor)"},
	{"Proyecto actual", "modo, fichas, recibo y riesgo de este directorio"},
	{"Sincronizar", "actualizar lo ya instalado a esta versión"},
	{"Desinstalar", "retirar bloques, hooks y skills (conserva lo tuyo)"},
	{"Restaurar último backup", "volver al estado anterior al último cambio"},
	{"Salir", ""},
}

const (
	minW = 60
	minH = 18
)

type applyDoneMsg struct {
	dir string
	err error
}

// Model es el modelo raíz (arquitectura Elm).
type Model struct {
	home, cwd string
	screen    screen
	op        op
	w, h      int
	cursor    int
	agents    []adapters.Adapter
	detected  []bool
	selected  []bool

	optHooks, optPerms bool
	force              bool
	plan               core.Plan
	diff               []string
	scroll             int
	planErr            error

	backupDir  string
	applyErr   error
	results    []doctor.Result
	doneNote   string
	proj       projectInfo
	restoreDir string
	restoreErr error
	quit       bool
}

// New crea el modelo; home es el HOME efectivo.
func New(home string) Model {
	cwd, _ := os.Getwd()
	m := Model{home: home, cwd: cwd, agents: adapters.All, w: 80, h: 30, optHooks: true, optPerms: true}
	for _, a := range m.agents {
		d := a.Detected()
		m.detected = append(m.detected, d)
		m.selected = append(m.selected, d) // solo los detectados vienen marcados
	}
	return m
}

// Accesores para pruebas.
func (m Model) Screen() int              { return int(m.screen) }
func (m Model) Plan() core.Plan          { return m.plan }
func (m Model) Quitting() bool           { return m.quit }
func (m Model) ApplyErr() error          { return m.applyErr }
func (m Model) BackupDir() string        { return m.backupDir }
func (m Model) Results() []doctor.Result { return m.results }
func (m Model) Op() int                  { return int(m.op) }

func (m Model) Init() tea.Cmd { return func() tea.Msg { uploadKitty(); return nil } }

func (m Model) selectedNames() []string {
	var n []string
	for i, a := range m.agents {
		if m.selected[i] {
			n = append(n, a.Name)
		}
	}
	return n
}

func (m Model) allNames() []string {
	var n []string
	for _, a := range m.agents {
		n = append(n, a.Name)
	}
	return n
}

func backupRoot(home string) string { return filepath.Join(home, ".v-ia", "backups") }

func (m *Model) buildPlan() {
	m.planErr, m.diff, m.scroll = nil, nil, 0
	var err error
	switch m.op {
	case opInstall:
		m.plan, err = lifecycle.Install(lifecycle.Options{Home: m.home, Agents: m.selectedNames(), Force: m.force, Hooks: m.optHooks, Permissions: m.optPerms})
	case opSync:
		m.plan, err = lifecycle.Sync(lifecycle.Options{Home: m.home, Agents: m.allNames(), Force: m.force, Hooks: true, Permissions: true})
	case opUninstall:
		m.plan, err = lifecycle.Uninstall(lifecycle.Options{Home: m.home, Agents: m.allNames()})
	}
	if err != nil {
		m.planErr = err
		return
	}
	for _, a := range m.plan.Actions {
		switch a.Status {
		case core.Unchanged:
			m.diff = append(m.diff, "= "+a.Path+" (sin cambios)")
		case core.Delete:
			m.diff = append(m.diff, "- "+a.Path+" (se elimina)")
		default:
			m.diff = append(m.diff, strings.Split(strings.TrimRight(a.Diff(), "\n"), "\n")...)
		}
	}
}

func applyCmd(p core.Plan, root string) tea.Cmd {
	return func() tea.Msg {
		d, err := core.Apply(p, root, time.Now())
		return applyDoneMsg{dir: d, err: err}
	}
}

func (m Model) goHome() Model {
	m.screen, m.cursor, m.force, m.scroll = scHome, 0, false, 0
	return m
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case applyDoneMsg:
		m.backupDir, m.applyErr = msg.dir, msg.err
		m.results = doctor.Run(m.home)
		m.doneNote = opDone[m.op]
		m.screen = scDone
	case tea.KeyMsg:
		return m.onKey(msg.String())
	}
	return m, nil
}

func (m Model) onKey(k string) (tea.Model, tea.Cmd) {
	if m.screen == scApply { // la escritura es atómica y con rollback; no se interrumpe
		return m, nil
	}
	if k == "ctrl+c" || (k == "q" && m.screen != scHome) {
		m.quit = true
		return m, tea.Quit
	}
	switch m.screen {
	case scHome:
		return m.onHome(k)
	case scAgents:
		switch k {
		case "up", "k":
			m.cursor = max(0, m.cursor-1)
		case "down", "j":
			m.cursor = min(len(m.agents)-1, m.cursor+1)
		case " ", "space":
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "enter":
			if len(m.selectedNames()) > 0 {
				m.screen, m.cursor = scComponents, 0
			}
		case "esc":
			m = m.goHome()
		}
	case scComponents:
		switch k {
		case "up", "k":
			m.cursor = max(0, m.cursor-1)
		case "down", "j":
			m.cursor = min(1, m.cursor+1)
		case " ", "space":
			if m.cursor == 0 {
				m.optHooks = !m.optHooks
			} else {
				m.optPerms = !m.optPerms
			}
		case "enter":
			m.buildPlan()
			m.screen = scReview
		case "esc":
			m.screen, m.cursor = scAgents, 0
		}
	case scReview:
		return m.onReview(k)
	case scDone, scStatus, scProject:
		switch k {
		case "enter", "esc":
			m = m.goHome()
		case "r":
			if m.screen == scStatus {
				m.results = doctor.Run(m.home)
			} else if m.screen == scProject {
				m.proj = loadProject(m.cwd, m.home)
			}
		case "q":
			m.quit = true
			return m, tea.Quit
		}
	case scRestore:
		switch k {
		case "enter":
			if m.restoreDir == "" {
				return m, nil
			}
			m.applyErr = core.Restore(m.restoreDir)
			m.backupDir = ""
			m.results = doctor.Run(m.home)
			m.doneNote = "Restaurado desde " + m.restoreDir
			m.screen = scDone
		case "esc":
			m = m.goHome()
		}
	}
	return m, nil
}

func (m Model) onHome(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "up", "k":
		m.cursor = max(0, m.cursor-1)
	case "down", "j":
		m.cursor = min(len(menu)-1, m.cursor+1)
	case "q":
		m.quit = true
		return m, tea.Quit
	case "enter":
		switch m.cursor {
		case 0:
			m.op, m.screen, m.cursor = opInstall, scAgents, 0
		case 1:
			m.results = doctor.Run(m.home)
			m.screen = scStatus
		case 2:
			m.proj = loadProject(m.cwd, m.home)
			m.screen = scProject
		case 3, 4:
			m.op = map[int]op{3: opSync, 4: opUninstall}[m.cursor]
			m.buildPlan()
			m.screen = scReview
		case 5:
			m.restoreDir, m.restoreErr = core.Latest(backupRoot(m.home))
			if m.restoreErr != nil {
				m.restoreDir = ""
			}
			m.screen = scRestore
		case 6:
			m.quit = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) onReview(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "up", "k":
		m.scroll = max(0, m.scroll-1)
	case "down", "j":
		m.scroll = min(max(0, len(m.diff)-m.viewLines()), m.scroll+1)
	case "f":
		if m.op != opUninstall {
			m.force = !m.force
			m.buildPlan()
		}
	case "esc":
		if m.op == opInstall {
			m.screen, m.cursor = scComponents, 0
		} else {
			m = m.goHome()
		}
	case "enter":
		if m.planErr != nil || len(m.plan.Conflicts()) > 0 {
			return m, nil
		}
		if len(m.plan.Changes()) == 0 {
			m.results = doctor.Run(m.home)
			m.backupDir, m.applyErr = "", nil
			m.doneNote = "Nada que cambiar."
			m.screen = scDone
			return m, nil
		}
		m.screen = scApply
		return m, applyCmd(m.plan, backupRoot(m.home))
	}
	return m, nil
}

// ---------------------------------------------------------------- vista

func (m Model) panelWidth() int { return min(m.w-4, 100) }

// headerHeight es lo que ocupan banner + lema; viewLines lo descuenta del área de contenido.
func (m Model) headerHeight() int {
	return lipgloss.Height(m.header())
}

func (m Model) viewLines() int { return max(3, m.h-m.headerHeight()-9) }

func (m Model) header() string {
	banner := Banner(pickBanner(m.w, m.h), m.w)
	tag := dimStyle.Render("ODD universal para tus agentes de IA  ·  ") + accentText.Render("v"+adapters.Version)
	return banner + "\n" + lipgloss.PlaceHorizontal(m.w, lipgloss.Center, tag) + "\n"
}

var installSteps = []string{"Agentes", "Componentes", "Revisión", "Aplicar", "Final"}

func (m Model) stepper() string {
	cur := map[screen]int{scAgents: 0, scComponents: 1, scReview: 2, scApply: 3, scDone: 4}[m.screen]
	if m.op != opInstall && (m.screen == scReview || m.screen == scApply || m.screen == scDone) {
		cur = map[screen]int{scReview: 2, scApply: 3, scDone: 4}[m.screen]
	}
	var parts []string
	for i, s := range installSteps {
		switch {
		case i < cur:
			parts = append(parts, okStyle.Render("✓ ")+dimStyle.Render(s))
		case i == cur:
			parts = append(parts, accentText.Render("● "+s))
		default:
			parts = append(parts, dimStyle.Render("○ "+s))
		}
	}
	return strings.Join(parts, dimStyle.Render("  ─  "))
}

func (m Model) title() string {
	switch m.screen {
	case scHome:
		return "Menú principal"
	case scStatus:
		return "Estado del sistema"
	case scProject:
		return "Proyecto actual"
	case scRestore:
		return "Restaurar backup"
	}
	name := map[op]string{opInstall: "Instalar", opSync: "Sincronizar", opUninstall: "Desinstalar"}[m.op]
	return name + "  " + m.stepper()
}

func (m Model) View() string {
	if m.w < minW || m.h < minH {
		return warnStyle.Render(fmt.Sprintf("! Terminal demasiado pequeña (%dx%d). Mínimo %dx%d.", m.w, m.h, minW, minH)) + "\n"
	}
	body, footer := m.content()
	inner := m.panelWidth() - 6
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		if lipgloss.Width(l) > inner {
			lines[i] = truncateVisible(l, inner)
		}
	}
	box := panel(m.panelWidth()).Render(accentText.Render(m.title()) + "\n\n" + strings.Join(lines, "\n"))
	box = lipgloss.PlaceHorizontal(m.w, lipgloss.Center, box)
	foot := lipgloss.PlaceHorizontal(m.w, lipgloss.Center, truncateVisible(footer, m.w))
	return m.header() + "\n" + box + "\n" + foot
}

// truncateVisible recorta por ancho visible sin romper secuencias de color.
func truncateVisible(s string, n int) string {
	if lipgloss.Width(s) <= n {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(n).Render(s)
}

func (m Model) content() (body, footer string) {
	var b strings.Builder
	switch m.screen {
	case scHome:
		for i, it := range menu {
			if i == m.cursor {
				b.WriteString(selStyle.Render("▸ "+it.label) + "\n")
				if it.desc != "" {
					b.WriteString("    " + dimStyle.Render(it.desc) + "\n")
				}
			} else {
				b.WriteString("  " + it.label + "\n")
			}
		}
		return strings.TrimRight(b.String(), "\n"), help("↑↓", "mover", "enter", "elegir", "q", "salir")
	case scAgents:
		b.WriteString("Elige los agentes a configurar:\n\n")
		for i, a := range m.agents {
			cur, box := "  ", "[ ]"
			if i == m.cursor {
				cur = selStyle.Render("▸ ")
			}
			if m.selected[i] {
				box = okStyle.Render("[x]")
			}
			how := dimStyle.Render("instrucciones")
			if a.Hooks {
				how = accentText.Render("bloqueo por hooks")
			}
			b.WriteString(fmt.Sprintf("%s%s %-12s %s%s\n", cur, box, a.Name, how, dimStyle.Render(detectedLabel(m.detected[i]))))
		}
		if len(m.selectedNames()) == 0 {
			b.WriteString("\n" + warnStyle.Render("! Marca al menos un agente para continuar."))
		}
		return strings.TrimRight(b.String(), "\n"), help("↑↓", "mover", "espacio", "marcar", "enter", "siguiente", "esc", "inicio")
	case scComponents:
		row := func(i int, on bool, label, desc string) {
			cur, box := "  ", "[ ]"
			if m.cursor == i {
				cur = selStyle.Render("▸ ")
			}
			if on {
				box = okStyle.Render("[x]")
			}
			b.WriteString(cur + box + " " + label + "\n      " + dimStyle.Render(desc) + "\n")
		}
		b.WriteString(okStyle.Render("  [x]") + " ODD — instrucciones y clasificación " + dimStyle.Render("(obligatorio)") + "\n\n")
		row(0, m.optHooks, "Hooks de bloqueo (Claude Code)", "impiden editar en modo lectura, entregar sin pedirlo y cerrar sin evidencia")
		row(1, m.optPerms, "Lista de archivos sensibles", "niega leer ~/.ssh, .env, *.pem, *.key y credenciales")
		b.WriteString("\n" + dimStyle.Render("  Siempre con Claude Code: skills (5) y subagente revisor v-ia-reviewer."))
		b.WriteString("\n" + dimStyle.Render("  Memoria: Engram "+map[bool]string{true: "detectado", false: "no detectado (opcional)"}[hasEngram()]))
		return b.String(), help("↑↓", "mover", "espacio", "alternar", "enter", "revisar", "esc", "atrás")
	case scReview:
		return m.viewReview()
	case scApply:
		return "Aplicando cambios…\n\n" + dimStyle.Render("backup previo · escritura atómica · rollback si algo falla"), dimStyle.Render("no se puede interrumpir")
	case scDone:
		return m.viewDone()
	case scStatus:
		b.WriteString(m.resultsView())
		return b.String(), help("r", "refrescar", "enter", "inicio", "q", "salir")
	case scProject:
		return m.viewProject()
	case scRestore:
		if m.restoreDir == "" {
			return warnStyle.Render("! No hay backups que restaurar."), help("esc", "inicio")
		}
		return "Se restaurarán los archivos al estado guardado en:\n\n  " + accentText.Render(m.restoreDir) + "\n\n" +
			dimStyle.Render("Los archivos que V-IA creó después se eliminan; los que modificó vuelven a su contenido anterior."), help("enter", "restaurar", "esc", "cancelar")
	}
	return "", ""
}

func hasEngram() bool {
	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		if st, err := os.Stat(filepath.Join(d, "engram")); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}

func (m Model) viewReview() (string, string) {
	if m.planErr != nil {
		return errStyle.Render("✗ " + m.planErr.Error()), help("esc", "volver", "q", "salir")
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Cambios previstos: %s  ·  conflictos: %s\n\n",
		accentText.Render(fmt.Sprint(len(m.plan.Changes()))), conflictCount(len(m.plan.Conflicts()))))
	end := min(len(m.diff), m.scroll+m.viewLines())
	for _, l := range m.diff[m.scroll:end] {
		b.WriteString(colorDiff(l) + "\n")
	}
	if len(m.diff) > m.viewLines() {
		b.WriteString(dimStyle.Render(fmt.Sprintf("— líneas %d-%d de %d —", m.scroll+1, end, len(m.diff))) + "\n")
	}
	for _, c := range m.plan.Conflicts() {
		b.WriteString(warnStyle.Render("! "+c.Path+": editado a mano; 'f' para sobrescribir (con backup)") + "\n")
	}
	keys := []string{"↑↓", "desplazar", "f", "forzar", "esc", "atrás"}
	if m.op == opUninstall {
		keys = []string{"↑↓", "desplazar", "esc", "atrás"}
	}
	if len(m.plan.Conflicts()) == 0 {
		keys = append([]string{"enter", "aplicar"}, keys...)
	}
	return strings.TrimRight(b.String(), "\n"), help(keys...)
}

func conflictCount(n int) string {
	if n > 0 {
		return warnStyle.Render(fmt.Sprint(n))
	}
	return okStyle.Render("0")
}

func (m Model) resultsView() string {
	var b strings.Builder
	for _, r := range m.results {
		b.WriteString(levelStyle(r.Level).Render(r.Level.Symbol()+" "+r.Msg) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m Model) viewDone() (string, string) {
	var b strings.Builder
	switch {
	case m.applyErr != nil:
		b.WriteString(errStyle.Render("✗ "+m.applyErr.Error()) + "\n\n")
	case m.backupDir != "":
		b.WriteString(okStyle.Render("✓ "+m.doneNote) + dimStyle.Render("  Backup: "+m.backupDir) + "\n\n")
	default:
		b.WriteString(okStyle.Render("✓ "+m.doneNote) + "\n\n")
	}
	b.WriteString(titleStyle.Render("Diagnóstico") + "\n" + m.resultsView() + "\n\n")
	b.WriteString(dimStyle.Render("Siguiente: v-ia task new <feature> · v-ia review check · v-ia restore latest --yes"))
	return b.String(), help("enter", "inicio", "q", "salir")
}

func (m Model) viewProject() (string, string) {
	p := m.proj
	row := func(k, v string) string { return fmt.Sprintf("%s %s\n", dimStyle.Render(fmt.Sprintf("%-10s", k)), v) }
	var b strings.Builder
	b.WriteString(row("Directorio", p.Dir))
	b.WriteString(row("Modo", accentText.Render(p.Mode)))
	b.WriteString(row("RDD", p.RDD))
	b.WriteString(row("Recibo", p.Receipt))
	b.WriteString(row("Riesgo", p.Risk))
	if len(p.Tasks) == 0 {
		b.WriteString(row("Fichas", dimStyle.Render("ninguna en odd/tasks/")))
	}
	for i, t := range p.Tasks {
		k := ""
		if i == 0 {
			k = "Fichas"
		}
		b.WriteString(row(k, t))
	}
	return strings.TrimRight(b.String(), "\n"), help("r", "refrescar", "enter", "inicio", "q", "salir")
}
