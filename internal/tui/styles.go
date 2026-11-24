package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"v-ia/internal/doctor"
)

// Paleta de marca: blanco y el violeta del logo. El estado nunca depende solo del color
// (✓ ! ✗ llevan símbolo), y lipgloss respeta NO_COLOR.
var (
	accent     = lipgloss.Color("#8B5CF6") // violeta legible sobre fondo oscuro
	titleStyle = lipgloss.NewStyle().Bold(true)
	dimStyle   = lipgloss.NewStyle().Faint(true)
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	accentText = lipgloss.NewStyle().Foreground(accent).Bold(true)
	selStyle   = lipgloss.NewStyle().Foreground(accent).Bold(true)
	keyStyle   = lipgloss.NewStyle().Foreground(accent)
)

func panel(w int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(0, 2).
		Width(w)
}

func levelStyle(l doctor.Level) lipgloss.Style {
	return [...]lipgloss.Style{okStyle, warnStyle, errStyle}[l]
}

func mark(ok bool) string {
	if ok {
		return okStyle.Render("✓")
	}
	return warnStyle.Render("!")
}

func detectedLabel(d bool) string {
	if d {
		return "  detectado"
	}
	return "  no detectado"
}

// help pinta "tecla acción" con la tecla resaltada: help("enter", "aplicar", "esc", "volver").
func help(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, keyStyle.Render(pairs[i])+" "+dimStyle.Render(pairs[i+1]))
	}
	return strings.Join(parts, dimStyle.Render(" · "))
}

func colorDiff(l string) string {
	switch {
	case strings.HasPrefix(l, "+"):
		return okStyle.Render(l)
	case strings.HasPrefix(l, "-"):
		return errStyle.Render(l)
	}
	return l
}

func truncate(s string, n int) string {
	r := []rune(s)
	if n < 4 || len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// Run abre la TUI.
func Run(home string) error {
	_, err := tea.NewProgram(New(home), tea.WithAltScreen()).Run()
	return err
}
