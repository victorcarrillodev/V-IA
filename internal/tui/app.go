// Package tui es la capa interactiva (Bubble Tea) sobre internal/core.
// Solo construye un Plan y llama a core; no contiene lógica de instalación.
package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"v-ia/internal/adapters"
	"v-ia/internal/core"
	"v-ia/internal/doctor"
)

type screen int

const (
	scWelcome screen = iota
	scAgents
	scComponents
	scReview
	scApply
	scDone
)

var screenNames = [...]string{"Bienvenida", "Agentes", "Componentes", "Revisión", "Aplicar", "Final"}

const (
	minW = 60
	minH = 16
)

type applyDoneMsg struct {
	dir string
	err error
}

// Model es el modelo raíz (arquitectura Elm).
type Model struct {
	home     string
	screen   screen
	w, h     int
	cursor   int
	agents   []adapters.Adapter
	detected []bool
	selected []bool

	force   bool
	plan    core.Plan
	diff    []string
	scroll  int
	planErr error

	backupDir string
	applyErr  error
	results   []doctor.Result
	quit      bool
}

// New crea el modelo; home es el HOME efectivo.
func New(home string) Model {
	m := Model{home: home, agents: adapters.All, w: 80, h: 24}
	for _, a := range m.agents {
		d := a.Detected()
		m.detected = append(m.detected, d)
		m.selected = append(m.selected, d) // solo los detectados vienen marcados
	}
	return m
}

// Screen y Selected se exponen para pruebas.
func (m Model) Screen() int              { return int(m.screen) }
func (m Model) Plan() core.Plan          { return m.plan }
func (m Model) Quitting() bool           { return m.quit }
func (m Model) ApplyErr() error          { return m.applyErr }
func (m Model) BackupDir() string        { return m.backupDir }
func (m Model) Results() []doctor.Result { return m.results }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) selectedNames() []string {
	var n []string
	for i, a := range m.agents {
		if m.selected[i] {
			n = append(n, a.Name)
		}
	}
	return n
}

func (m *Model) buildPlan() {
	m.planErr, m.diff, m.scroll = nil, nil, 0
	ts, err := adapters.Targets(m.home, m.selectedNames())
	if err == nil {
		m.plan, err = core.BuildPlan(ts, m.force)
	}
	if err != nil {
		m.planErr = err
		return
	}
	for _, a := range m.plan.Actions {
		if a.Status == core.Unchanged {
			m.diff = append(m.diff, "= "+a.Path+" (sin cambios)")
			continue
		}
		m.diff = append(m.diff, strings.Split(strings.TrimRight(a.Diff(), "\n"), "\n")...)
	}
}

func applyCmd(p core.Plan, root string) tea.Cmd {
	return func() tea.Msg {
		d, err := core.Apply(p, root, time.Now())
		return applyDoneMsg{dir: d, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case applyDoneMsg:
		m.backupDir, m.applyErr = msg.dir, msg.err
		m.results = doctor.Run(m.home)
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
	switch k {
	case "ctrl+c", "q":
		m.quit = true
		return m, tea.Quit
	}
	switch m.screen {
	case scWelcome:
		if k == "enter" {
			m.screen = scAgents
			m.cursor = 0
		}
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
				m.screen = scComponents
			}
		case "esc":
			m.screen = scWelcome
		}
	case scComponents:
		switch k {
		case "enter":
			m.buildPlan()
			m.screen = scReview
		case "esc":
			m.screen = scAgents
		}
	case scReview:
		switch k {
		case "up", "k":
			m.scroll = max(0, m.scroll-1)
		case "down", "j":
			m.scroll = min(max(0, len(m.diff)-m.viewLines()), m.scroll+1)
		case "f":
			m.force = !m.force
			m.buildPlan()
		case "esc":
			m.screen = scComponents
		case "enter":
			if m.planErr != nil || len(m.plan.Conflicts()) > 0 {
				return m, nil
			}
			if len(m.plan.Changes()) == 0 {
				m.results = doctor.Run(m.home)
				m.screen = scDone
				return m, nil
			}
			m.screen = scApply
			return m, applyCmd(m.plan, filepath.Join(m.home, ".v-ia", "backups"))
		}
	case scDone:
		if k == "enter" {
			m.quit = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) viewLines() int { return max(3, m.h-9) }

func (m Model) header() string {
	return titleStyle.Render(fmt.Sprintf("V-IA %s", adapters.Version)) +
		dimStyle.Render(fmt.Sprintf("  ·  %d/%d %s", int(m.screen)+1, len(screenNames), screenNames[m.screen]))
}

func (m Model) View() string {
	if m.w < minW || m.h < minH {
		return warnStyle.Render(fmt.Sprintf("! Terminal demasiado pequeña (%dx%d). Mínimo %dx%d.", m.w, m.h, minW, minH)) + "\n"
	}
	var b strings.Builder
	b.WriteString(m.header() + "\n\n")
	switch m.screen {
	case scWelcome:
		b.WriteString("ODD universal para tus agentes de IA.\n\nAgentes soportados:\n")
		for i, a := range m.agents {
			b.WriteString("  " + mark(m.detected[i]) + " " + a.Name + dimStyle.Render(detectedLabel(m.detected[i])) + "\n")
		}
		b.WriteString("\n" + help("enter siguiente", "q salir"))
	case scAgents:
		b.WriteString("Elige los agentes a configurar:\n\n")
		for i, a := range m.agents {
			cur := "  "
			if i == m.cursor {
				cur = "> "
			}
			box := "[ ]"
			if m.selected[i] {
				box = "[x]"
			}
			b.WriteString(cur + box + " " + a.Name + dimStyle.Render(detectedLabel(m.detected[i])) + "\n")
		}
		if len(m.selectedNames()) == 0 {
			b.WriteString("\n" + warnStyle.Render("! Marca al menos un agente para continuar."))
		}
		b.WriteString("\n" + help("↑/↓ mover", "espacio marcar", "enter siguiente", "esc volver", "q salir"))
	case scComponents:
		b.WriteString("Componentes:\n\n")
		b.WriteString("  [x] ODD — instrucciones y clasificación " + dimStyle.Render("(obligatorio)") + "\n")
		for _, c := range []string{"Engram (memoria)", "Skills", "RDD (revisión)", "Subagentes"} {
			b.WriteString("  [ ] " + dimStyle.Render(c+" — próximamente") + "\n")
		}
		b.WriteString("\n" + help("enter revisar cambios", "esc volver", "q salir"))
	case scReview:
		b.WriteString(m.viewReview())
	case scApply:
		b.WriteString("Aplicando cambios (backup previo, escritura atómica, rollback si falla)…\n")
	case scDone:
		b.WriteString(m.viewDone())
	}
	return b.String()
}

func (m Model) viewReview() string {
	var b strings.Builder
	if m.planErr != nil {
		return errStyle.Render("✗ "+m.planErr.Error()) + "\n\n" + help("esc volver", "q salir")
	}
	b.WriteString(fmt.Sprintf("Cambios previstos: %d · conflictos: %d\n\n", len(m.plan.Changes()), len(m.plan.Conflicts())))
	end := min(len(m.diff), m.scroll+m.viewLines())
	for _, l := range m.diff[m.scroll:end] {
		b.WriteString(colorDiff(truncate(l, m.w-2)) + "\n")
	}
	for _, c := range m.plan.Conflicts() {
		b.WriteString(warnStyle.Render("! "+c.Path+": bloque editado a mano; 'f' para sobrescribir (con backup)") + "\n")
	}
	keys := []string{"↑/↓ desplazar", "f forzar", "esc volver", "q salir"}
	if len(m.plan.Conflicts()) == 0 {
		keys = append([]string{"enter aplicar"}, keys...)
	}
	return b.String() + "\n" + help(keys...)
}

func (m Model) viewDone() string {
	var b strings.Builder
	if m.applyErr != nil {
		b.WriteString(errStyle.Render("✗ "+m.applyErr.Error()) + "\n\n")
	} else if m.backupDir != "" {
		b.WriteString(okStyle.Render("✓ Aplicado.") + dimStyle.Render(" Backup: "+m.backupDir) + "\n\n")
	} else {
		b.WriteString(okStyle.Render("✓ Nada que cambiar.") + "\n\n")
	}
	b.WriteString("Diagnóstico:\n")
	for _, r := range m.results {
		b.WriteString("  " + levelStyle(r.Level).Render(r.Level.Symbol()+" "+truncate(r.Msg, m.w-6)) + "\n")
	}
	b.WriteString("\nSiguiente: " + dimStyle.Render("v-ia restore latest --yes (deshacer) · v-ia task new <feature>") + "\n")
	return b.String() + "\n" + help("enter salir")
}
