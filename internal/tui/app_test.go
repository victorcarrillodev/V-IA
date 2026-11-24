package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"v-ia/internal/extras"
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
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg")) // nunca escribir en la config real
	t.Setenv("CODEX_HOME", filepath.Join(home, "codex"))
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
	if m.Screen() != int(scReview) || len(m.Plan().Changes()) < 2 {
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
	if m.Quitting() || m.Screen() != int(scHome) {
		t.Fatalf("enter en Final debe volver al menú, no salir (pantalla=%d)", m.Screen())
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

func TestMenuNavegacionYSalir(t *testing.T) {
	m, _ := newModel(t)
	m = send(t, m, "j")
	m = send(t, m, "j")
	if m.cursor != 2 {
		t.Fatalf("cursor=%d", m.cursor)
	}
	m = send(t, m, "k")
	if m.cursor != 1 {
		t.Fatalf("cursor=%d", m.cursor)
	}
	for i := 0; i < 10; i++ {
		m = send(t, m, "down")
	}
	if m.cursor != len(menu)-1 {
		t.Fatalf("el cursor no debe pasarse del final: %d", m.cursor)
	}
	m = send(t, m, "enter") // Salir
	if !m.Quitting() {
		t.Fatal("la opción Salir debe salir")
	}
}

func goTo(t *testing.T, m Model, idx int) Model {
	t.Helper()
	for i := 0; i < idx; i++ {
		m = send(t, m, "j")
	}
	return send(t, m, "enter")
}

func TestEstadoYProyectoSonSoloLectura(t *testing.T) {
	m, home := newModel(t)
	m = goTo(t, m, 1)
	if m.Screen() != int(scStatus) || len(m.Results()) == 0 {
		t.Fatalf("pantalla=%d resultados=%d", m.Screen(), len(m.Results()))
	}
	if !strings.Contains(m.View(), "Estado del sistema") {
		t.Fatal("falta el título")
	}
	m = send(t, m, "esc")
	m.cwd = t.TempDir()
	m = goTo(t, m, 2)
	v := m.View()
	if m.Screen() != int(scProject) || !strings.Contains(v, "Modo") || !strings.Contains(v, "Recibo") || !strings.Contains(v, "Riesgo") {
		t.Fatalf("panel de proyecto incompleto:\n%s", v)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude")); err == nil {
		t.Fatal("estado y proyecto no deben escribir nada")
	}
}

func TestInstalarSincronizarDesinstalarYRestaurar(t *testing.T) {
	m, home := newModel(t)
	for _, k := range []string{"enter", "enter", "enter", "enter"} { // instalar completo
		m = send(t, m, k)
	}
	claude := filepath.Join(home, ".claude", "CLAUDE.md")
	if _, err := os.Stat(claude); err != nil {
		t.Fatalf("no se instaló: %v", err)
	}
	m = send(t, m, "enter") // Final → menú
	m = goTo(t, m, 3)       // Sincronizar: ya está al día
	if m.Screen() != int(scReview) || len(m.Plan().Changes()) != 0 || m.Op() != int(opSync) {
		t.Fatalf("sync tras instalar debe estar al día: pantalla=%d cambios=%d", m.Screen(), len(m.Plan().Changes()))
	}
	m = send(t, m, "enter")
	if m.Screen() != int(scDone) || !strings.Contains(m.View(), "Nada que cambiar") {
		t.Fatal("sin cambios debe decir 'Nada que cambiar'")
	}
	m = send(t, m, "enter") // menú
	m = goTo(t, m, 4)       // Desinstalar
	if m.Op() != int(opUninstall) || len(m.Plan().Changes()) < 2 {
		t.Fatalf("uninstall debe planear retiradas: %d", len(m.Plan().Changes()))
	}
	m = send(t, m, "enter")
	if m.ApplyErr() != nil || m.Screen() != int(scDone) {
		t.Fatalf("uninstall falló: %v", m.ApplyErr())
	}
	if b, _ := os.ReadFile(claude); strings.Contains(string(b), "v-ia:managed") {
		t.Fatal("el bloque debía retirarse")
	}
	m = send(t, m, "enter") // menú
	m = goTo(t, m, 5)       // Restaurar
	if m.Screen() != int(scRestore) || m.restoreDir == "" {
		t.Fatal("debe ofrecer el último backup")
	}
	m = send(t, m, "enter")
	if m.ApplyErr() != nil || m.Screen() != int(scDone) {
		t.Fatalf("restaurar falló: %v", m.ApplyErr())
	}
	if b, _ := os.ReadFile(claude); !strings.Contains(string(b), "v-ia:managed begin id=odd") {
		t.Fatal("restaurar debe devolver el bloque")
	}
}

func TestRestaurarSinBackups(t *testing.T) {
	m, _ := newModel(t)
	m = goTo(t, m, 5)
	if m.Screen() != int(scRestore) || !strings.Contains(m.View(), "No hay backups") {
		t.Fatal("debe avisar que no hay backups")
	}
	m = send(t, m, "enter")
	if m.Screen() != int(scRestore) {
		t.Fatal("enter sin backup no debe hacer nada")
	}
}

func TestComponentesAlternanHooksYPermisos(t *testing.T) {
	m, _ := newModel(t)
	m = send(t, m, "enter")
	m = send(t, m, "enter") // componentes
	m = send(t, m, " ")     // hooks off
	m = send(t, m, "j")
	m = send(t, m, " ") // permisos off
	if m.optHooks || m.optPerms {
		t.Fatalf("hooks=%v perms=%v", m.optHooks, m.optPerms)
	}
	m = send(t, m, "enter")
	for _, a := range m.Plan().Actions {
		if strings.HasSuffix(a.Path, "settings.json") {
			t.Fatal("sin hooks ni permisos no se toca settings.json")
		}
	}
}

func TestComponentesOpcionalesAparecenEnLaRevision(t *testing.T) {
	old := extras.LookPath
	extras.LookPath = func(b string) (string, error) { return "/fake/" + b, nil }
	t.Cleanup(func() { extras.LookPath = old })
	m, _ := newModel(t)
	m = send(t, m, "enter")
	m = send(t, m, "enter") // componentes
	for i := 0; i < 3; i++ {
		m = send(t, m, "j")
	}
	m = send(t, m, " ") // cursor 3 = Context7
	if !m.optExtra[1] {
		t.Fatal("Context7 debía quedar marcado")
	}
	m = send(t, m, "enter")
	if m.planErr != nil {
		t.Fatalf("planErr=%v", m.planErr)
	}
	if !strings.Contains(strings.Join(m.diff, "\n"), "$ claude mcp add --scope user context7") {
		t.Fatalf("la revisión debe mostrar el comando de Context7:\n%v", m.diff)
	}
}

func TestComponentesOpcionalesFaltanRequisitos(t *testing.T) {
	old := extras.LookPath
	extras.LookPath = func(string) (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { extras.LookPath = old })
	m, _ := newModel(t)
	m = send(t, m, "enter")
	m = send(t, m, "enter")
	for i := 0; i < 4; i++ {
		m = send(t, m, "j")
	}
	m = send(t, m, " ") // CodeGraph
	m = send(t, m, "enter")
	if m.planErr == nil || !strings.Contains(m.planErr.Error(), "codegraph") {
		t.Fatalf("debía fallar por requisitos: %v", m.planErr)
	}
}
