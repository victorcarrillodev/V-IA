package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// send aplica una tecla y ejecuta el tea.Cmd resultante (como haría el runtime).
func send(t *testing.T, m Model, k string) Model {
	t.Helper()
	nm, cmd := m.Update(key(k))
	m = nm.(Model)
	if cmd != nil {
		if msg := cmd(); msg != nil {
			if _, isQuit := msg.(tea.QuitMsg); !isQuit {
				nm, _ = m.Update(msg)
				m = nm.(Model)
			}
		}
	}
	return m
}

func newModel(t *testing.T) (Model, string) {
	home := t.TempDir()
	m := New(home)
	for i := range m.selected { // independiente de qué agentes haya en el PC de test
		m.selected[i] = true
	}
	return m, home
}

func TestFlujoCompleto(t *testing.T) {
	m, home := newModel(t)
	for _, k := range []string{"enter", "enter", "enter"} { // welcome → agentes → componentes → revisión
		m = send(t, m, k)
	}
	if m.Screen() != int(scReview) || len(m.Plan().Changes()) != 1 {
		t.Fatalf("pantalla=%d cambios=%d", m.Screen(), len(m.Plan().Changes()))
	}
	target := filepath.Join(home, ".claude", "CLAUDE.md")
	if _, err := os.Stat(target); err == nil {
		t.Fatal("la pantalla de revisión no debe escribir")
	}
	m = send(t, m, "enter") // aplicar
	if m.Screen() != int(scDone) || m.ApplyErr() != nil || m.BackupDir() == "" {
		t.Fatalf("pantalla=%d err=%v", m.Screen(), m.ApplyErr())
	}
	b, err := os.ReadFile(target)
	if err != nil || !strings.Contains(string(b), "v-ia:managed begin id=odd") {
		t.Fatalf("no se instaló el bloque: %v", err)
	}
	if len(m.Results()) == 0 {
		t.Fatal("falta el resultado de doctor")
	}
	m = send(t, m, "enter")
	if !m.Quitting() {
		t.Fatal("enter en Final debe salir")
	}
}

func TestEscVuelveAtras(t *testing.T) {
	m, _ := newModel(t)
	m = send(t, m, "enter")
	m = send(t, m, "enter")
	m = send(t, m, "esc")
	if m.Screen() != int(scAgents) {
		t.Fatalf("pantalla=%d", m.Screen())
	}
}

func TestSinAgentesNoAvanza(t *testing.T) {
	m, _ := newModel(t)
	for i := range m.selected {
		m.selected[i] = false
	}
	m = send(t, m, "enter")
	m = send(t, m, "enter")
	if m.Screen() != int(scAgents) {
		t.Fatalf("avanzó sin agentes: %d", m.Screen())
	}
	m = send(t, m, " ")
	m = send(t, m, "enter")
	if m.Screen() != int(scComponents) {
		t.Fatalf("no avanzó tras marcar: %d", m.Screen())
	}
}

func TestConflictoBloqueaYForceDesbloquea(t *testing.T) {
	m, home := newModel(t)
	target := filepath.Join(home, ".claude", "CLAUDE.md")
	os.MkdirAll(filepath.Dir(target), 0o755)
	m2, _ := newModel(t)
	m2.home = home
	for _, k := range []string{"enter", "enter", "enter", "enter"} {
		m2 = send(t, m2, k)
	}
	b, _ := os.ReadFile(target)
	os.WriteFile(target, []byte(strings.Replace(string(b), "Autorizar", "EDITADO", 1)), 0o644)

	m = New(home)
	for i := range m.selected {
		m.selected[i] = true
	}
	for _, k := range []string{"enter", "enter", "enter"} {
		m = send(t, m, k)
	}
	if len(m.Plan().Conflicts()) != 1 {
		// el bloque coincide en versión: edición manual sin cambio de versión sigue siendo conflicto solo si cambia el cuerpo
		t.Fatalf("conflictos=%d", len(m.Plan().Conflicts()))
	}
	m = send(t, m, "enter")
	if m.Screen() != int(scReview) {
		t.Fatal("con conflicto no debe aplicar")
	}
	m = send(t, m, "f")
	if len(m.Plan().Conflicts()) != 0 {
		t.Fatal("force debe resolver el conflicto")
	}
}

func TestTerminalPequena(t *testing.T) {
	m, _ := newModel(t)
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	if !strings.Contains(nm.(Model).View(), "demasiado pequeña") {
		t.Fatal("falta el aviso de tamaño")
	}
}

func TestQSale(t *testing.T) {
	m, _ := newModel(t)
	m = send(t, m, "q")
	if !m.Quitting() {
		t.Fatal("q debe salir")
	}
}
