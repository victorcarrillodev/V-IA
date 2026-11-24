package tui

import (
	"os"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func plain(s string) string { return ansiRe.ReplaceAllString(s, "") }

func TestBannerMapasConsistentes(t *testing.T) {
	bannerOnce.Do(loadBanners)
	want := map[bannerKind][2]int{bannerLarge: {76, 16}, bannerSmall: {46, 10}}
	for k, dim := range want {
		rows := bannerCache[k]
		if len(rows) != dim[1] {
			t.Fatalf("banner %d: %d filas, quería %d", k, len(rows), dim[1])
		}
		for i, r := range rows {
			if len(r) != dim[0] {
				t.Fatalf("banner %d fila %d: ancho %d, quería %d", k, i, len(r), dim[0])
			}
			if strings.Trim(r, ".WP") != "" {
				t.Fatalf("banner %d fila %d: clases inválidas %q", k, i, r)
			}
		}
		all := strings.Join(rows, "")
		if !strings.Contains(all, "P") || !strings.Contains(all, "W") {
			t.Fatalf("banner %d debe tener blanco y violeta", k)
		}
	}
}

func TestBannerConColorUsaBlancoYVioleta(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	out := Banner(bannerLarge, 100)
	if !strings.Contains(out, "109;40;255") { // #6D28FF
		t.Fatal("falta el violeta del logo")
	}
	if !strings.Contains(out, "255;255;255") {
		t.Fatal("falta el blanco del logo")
	}
	if got := lipgloss.Height(out); got != 8 {
		t.Fatalf("alto=%d, quería 8 filas", got)
	}
}

func TestBannerSinColorSoloFormas(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	out := Banner(bannerSmall, 60)
	if ansiRe.MatchString(out) {
		t.Fatal("sin color no debe haber secuencias ANSI")
	}
	if !strings.ContainsAny(out, "█▀▄▒") || !strings.Contains(out, "▒") {
		t.Fatal("sin color el violeta debe distinguirse con sombreado")
	}
}

func TestPickBanner(t *testing.T) {
	cases := []struct {
		w, h int
		k    bannerKind
	}{{120, 50, bannerLarge}, {80, 30, bannerSmall}, {60, 24, bannerSmall}, {60, 20, bannerCompact}, {40, 50, bannerCompact}}
	for _, c := range cases {
		if got := pickBanner(c.w, c.h); got != c.k {
			t.Errorf("%dx%d: %d, quería %d", c.w, c.h, got, c.k)
		}
	}
}

// La vista nunca debe pasarse del terminal, en ninguna pantalla ni tamaño.
func TestVistaCabeEnElTerminal(t *testing.T) {
	sizes := [][2]int{{60, 18}, {80, 24}, {80, 30}, {100, 40}, {140, 50}}
	for _, sz := range sizes {
		m, _ := newModel(t)
		m.cwd = t.TempDir()
		nm, _ := m.Update(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		m = nm.(Model)
		for _, sc := range []screen{scHome, scAgents, scComponents, scReview, scDone, scStatus, scProject, scRestore} {
			mm := m
			mm.screen = sc
			switch sc {
			case scReview:
				mm.buildPlan()
			case scDone:
				mm.doneNote = "Aplicado."
				mm.results = nil
			case scProject:
				mm.proj = loadProject(mm.cwd, mm.home)
			}
			v := plain(mm.View())
			lines := strings.Split(strings.TrimRight(v, "\n"), "\n")
			if len(lines) > sz[1] {
				t.Errorf("%dx%d pantalla %d: %d líneas > %d\n%s", sz[0], sz[1], sc, len(lines), sz[1], v)
			}
			for _, l := range lines {
				if w := lipgloss.Width(l); w > sz[0] {
					t.Errorf("%dx%d pantalla %d: línea de %d columnas: %q", sz[0], sz[1], sc, w, l)
					break
				}
			}
		}
	}
}

// DUMP_TUI=1 go test ./internal/tui -run Dump -v  → imprime las pantallas para revisarlas a ojo.
func TestDumpPantallas(t *testing.T) {
	if os.Getenv("DUMP_TUI") == "" {
		t.Skip("solo con DUMP_TUI=1")
	}
	m, _ := newModel(t)
	m.cwd = t.TempDir()
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = nm.(Model)
	for _, sc := range []screen{scHome, scAgents, scComponents, scStatus, scProject} {
		mm := m
		mm.screen = sc
		mm.results = nil
		if sc == scProject {
			mm.proj = loadProject(mm.cwd, mm.home)
		}
		t.Logf("\n=== pantalla %d ===\n%s", sc, plain(mm.View()))
	}
}
