package tui

import (
	"embed"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

// Los mapas se generan desde docs/assets/logo.png con scripts/gen-banner.py.
// Cada celda de terminal = 1 px de ancho x 2 de alto (medios bloques ▀▄█).
//
//go:embed assets/banner_large.txt assets/banner_small.txt
var bannerFS embed.FS

type bannerKind int

const (
	bannerLarge bannerKind = iota
	bannerSmall
	bannerCompact
)

var (
	colWhite  = lipgloss.Color("#FFFFFF")
	colPurple = lipgloss.Color("#6D28FF") // el violeta del logo
)

var (
	bannerOnce  sync.Once
	bannerCache = map[bannerKind][]string{}
)

func loadBanners() {
	for k, f := range map[bannerKind]string{bannerLarge: "assets/banner_large.txt", bannerSmall: "assets/banner_small.txt"} {
		b, err := bannerFS.ReadFile(f)
		if err != nil {
			continue
		}
		bannerCache[k] = strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	}
}

// pickBanner elige el tamaño más grande que cabe sin comerse el contenido.
func pickBanner(w, h int) bannerKind {
	switch {
	case w >= 86 && h >= 36:
		return bannerLarge
	case w >= 54 && h >= 24:
		return bannerSmall
	}
	return bannerCompact
}

// colorEnabled es falso con NO_COLOR o sin TTY: entonces el logo se dibuja solo con formas.
func colorEnabled() bool {
	return lipgloss.NewStyle().Foreground(colWhite).Render("x") != "x"
}

func clsColor(c byte) lipgloss.Color {
	if c == 'P' {
		return colPurple
	}
	return colWhite
}

// cell devuelve el carácter y el estilo de un par de píxeles (arriba, abajo).
func cell(top, bot byte, color bool) (string, lipgloss.Style) {
	s := lipgloss.NewStyle()
	switch {
	case top == '.' && bot == '.':
		return " ", s
	case !color:
		// sin color: el violeta se distingue con sombreado
		shade := top == 'P' || bot == 'P'
		switch {
		case top != '.' && bot != '.':
			if shade {
				return "▒", s
			}
			return "█", s
		case top != '.':
			return "▀", s
		}
		return "▄", s
	case top == bot:
		return "█", s.Foreground(clsColor(top))
	case bot == '.':
		return "▀", s.Foreground(clsColor(top))
	case top == '.':
		return "▄", s.Foreground(clsColor(bot))
	}
	return "▀", s.Foreground(clsColor(top)).Background(clsColor(bot))
}

func renderMap(rows []string, color bool) string {
	var out []string
	for y := 0; y+1 < len(rows); y += 2 {
		var sb strings.Builder
		var run strings.Builder
		var runStyle lipgloss.Style
		have := false
		flush := func() {
			if have {
				sb.WriteString(runStyle.Render(run.String()))
				run.Reset()
				have = false
			}
		}
		top, bot := rows[y], rows[y+1]
		for x := 0; x < len(top) && x < len(bot); x++ {
			ch, st := cell(top[x], bot[x], color)
			if have && !sameStyle(st, runStyle) {
				flush()
			}
			if !have {
				runStyle, have = st, true
			}
			run.WriteString(ch)
		}
		flush()
		out = append(out, sb.String())
	}
	return strings.Join(out, "\n")
}

func sameStyle(a, b lipgloss.Style) bool {
	return a.GetForeground() == b.GetForeground() && a.GetBackground() == b.GetBackground()
}

// Banner dibuja el logo V-AI centrado en el ancho dado.
func Banner(k bannerKind, width int) string {
	var art string
	if k == bannerCompact {
		w := lipgloss.NewStyle().Bold(true).Foreground(colWhite)
		p := lipgloss.NewStyle().Bold(true).Foreground(colPurple)
		art = w.Render("V") + p.Render("▰") + w.Render("AI")
	} else {
		bannerOnce.Do(loadBanners)
		rows := bannerCache[k]
		if len(rows) == 0 {
			return ""
		}
		art = renderMap(rows, colorEnabled())
	}
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, art)
}
