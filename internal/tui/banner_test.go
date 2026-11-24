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
	want := map[bannerKind][2]int{bannerLarge: {104, 24}, bannerSmall: {64, 16}}
	for k, dim := range want {
		rows := bannerCache[k]
		if len(rows) != dim[1] {
			t.Fatalf("banner %d: %d filas, quería %d", k, len(rows), dim[1])
		}
		on := 0
		for i, r := range rows {
			if len(r) != dim[0] {
				t.Fatalf("banner %d fila %d: ancho %d, quería %d", k, i, len(r), dim[0])
			}
			for _, p := range r {
				if p.on {
					on++
				}
			}
		}
		if on < dim[0]*dim[1]/6 {
			t.Fatalf("banner %d casi vacío: %d píxeles", k, on)
		}
	}
}

func TestBannerConColorUsaColorVerdadero(t *testing.T) {
	t.Setenv("V_IA_IMAGES", "0")
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	out := Banner(bannerLarge, 120)
	if !strings.Contains(out, "38;2;") || !strings.Contains(out, "48;2;") {
		t.Fatal("el banner debe usar color verdadero (fg y bg)")
	}
	if got := lipgloss.Height(out); got != 12 {
		t.Fatalf("alto=%d, quería 12 filas", got)
	}
}

func TestBannerSinColorSoloFormas(t *testing.T) {
	t.Setenv("V_IA_IMAGES", "0")
	lipgloss.SetColorProfile(termenv.Ascii)
	out := Banner(bannerSmall, 60)
	if ansiRe.MatchString(out) {
		t.Fatal("sin color no debe haber secuencias ANSI")
	}
	if !strings.ContainsAny(out, "█▀▄▒▓░") {
		t.Fatal("sin color el logo debe dibujarse con bloques")
	}
}

func TestPickBanner(t *testing.T) {
	cases := []struct {
		w, h int
		k    bannerKind
	}{{120, 50, bannerLarge}, {80, 30, bannerSmall}, {70, 28, bannerSmall}, {60, 24, bannerCompact}, {40, 50, bannerCompact}}
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

func TestBannerKittyPlaceholders(t *testing.T) {
	t.Setenv("V_IA_IMAGES", "1")
	out := Banner(bannerLarge, 120)
	if got := strings.Count(out, "\U0010EEEE"); got != 104*12 {
		t.Fatalf("placeholders=%d, quería %d", got, 104*12)
	}
	if !strings.Contains(out, "38;5;201m") {
		t.Fatal("falta el id de imagen en el color")
	}
	if up := kittyUpload(); !strings.HasPrefix(up, "\x1b_Ga=T,f=100,U=1") || !strings.Contains(up, "i=201") || !strings.Contains(up, "i=202") {
		t.Fatal("transmisión kitty inválida")
	}
}
