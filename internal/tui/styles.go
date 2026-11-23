package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"v-ia/internal/doctor"
)

// lipgloss respeta NO_COLOR; el estado nunca depende solo del color (✓ ! ✗).
var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	dimStyle   = lipgloss.NewStyle().Faint(true)
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

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

func help(keys ...string) string { return dimStyle.Render(strings.Join(keys, " · ")) }

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
