package tui

import (
	"embed"
	"fmt"
	"image/png"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

// Los PNG se generan desde docs/assets/logo-neon.png con scripts/gen-banner.py.
// Cada celda de terminal = 1 px de ancho x 2 de alto (medios bloques ▀▄█) con color verdadero;
// lipgloss degrada solo a 256/16 colores según el terminal. El negro puro es fondo.
//
//go:embed assets/banner_large.png assets/banner_small.png
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

type pixel struct {
	r, g, b uint8
	on      bool
}

var (
	bannerOnce  sync.Once
	bannerCache = map[bannerKind][][]pixel{}
)

func loadBanners() {
	for k, f := range map[bannerKind]string{bannerLarge: "assets/banner_large.png", bannerSmall: "assets/banner_small.png"} {
		fh, err := bannerFS.Open(f)
		if err != nil {
			continue
		}
		img, err := png.Decode(fh)
		fh.Close()
		if err != nil {
			continue
		}
		bd := img.Bounds()
		rows := make([][]pixel, bd.Dy())
		for y := range rows {
			rows[y] = make([]pixel, bd.Dx())
			for x := range rows[y] {
				r, g, b, _ := img.At(bd.Min.X+x, bd.Min.Y+y).RGBA()
				p := pixel{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), false}
				p.on = p.r|p.g|p.b != 0
				rows[y][x] = p
			}
		}
		bannerCache[k] = rows
	}
}

// pickBanner elige el tamaño más grande que cabe sin comerse el contenido.
func pickBanner(w, h int) bannerKind {
	switch {
	case w >= 112 && h >= 40:
		return bannerLarge
	case w >= 70 && h >= 28:
		return bannerSmall
	}
	return bannerCompact
}

// colorEnabled es falso con NO_COLOR o sin TTY: entonces el logo se dibuja solo con formas.
func colorEnabled() bool {
	return lipgloss.NewStyle().Foreground(colWhite).Render("x") != "x"
}

func (p pixel) color() lipgloss.Color {
	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", p.r, p.g, p.b))
}

// lum es el brillo percibido (0-255).
func (p pixel) lum() int { return (int(p.r)*299 + int(p.g)*587 + int(p.b)*114) / 1000 }

// shade elige un bloque según el brillo, para terminales sin color.
func shade(p pixel) string {
	switch l := p.lum(); {
	case l > 170:
		return "█"
	case l > 110:
		return "▓"
	case l > 60:
		return "▒"
	}
	return "░"
}

// cell devuelve el carácter y el estilo de un par de píxeles (arriba, abajo).
func cell(top, bot pixel, color bool) (string, lipgloss.Style) {
	s := lipgloss.NewStyle()
	switch {
	case !top.on && !bot.on:
		return " ", s
	case !color:
		if top.on && bot.on {
			if top.lum() >= bot.lum() {
				return shade(top), s
			}
			return shade(bot), s
		}
		if top.on {
			return "▀", s
		}
		return "▄", s
	case !bot.on:
		return "▀", s.Foreground(top.color())
	case !top.on:
		return "▄", s.Foreground(bot.color())
	}
	return "▀", s.Foreground(top.color()).Background(bot.color())
}

func renderMap(rows [][]pixel, color bool) string {
	var out []string
	for y := 0; y+1 < len(rows); y += 2 {
		var sb strings.Builder
		top, bot := rows[y], rows[y+1]
		for x := 0; x < len(top) && x < len(bot); x++ {
			ch, st := cell(top[x], bot[x], color)
			sb.WriteString(st.Render(ch))
		}
		out = append(out, sb.String())
	}
	return strings.Join(out, "\n")
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
