package tui

import (
	"embed"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// Protocolo de gráficos de kitty (Ghostty, kitty): el terminal dibuja la imagen real del logo.
// Se transmite una vez y la vista solo coloca "placeholders" Unicode, así que sobrevive a los
// redibujados de bubbletea. En otros terminales se usa el banner de medios bloques.
//
//go:embed assets/banner_kitty.png
var kittyFS embed.FS

const (
	kittyIDLarge = 201 // el id va en el color de primer plano (0-255)
	kittyIDSmall = 202
)

type kittyDims struct{ id, cols, rows int }

var kittySizes = map[bannerKind]kittyDims{
	bannerLarge: {kittyIDLarge, 104, 12},
	bannerSmall: {kittyIDSmall, 64, 8},
}

// kittyEnabled detecta terminales con protocolo kitty y placeholders. V_IA_IMAGES=0 lo desactiva.
func kittyEnabled() bool {
	switch os.Getenv("V_IA_IMAGES") {
	case "0":
		return false
	case "1":
		return true
	}
	if os.Getenv("TMUX") != "" || os.Getenv("STY") != "" {
		return false
	}
	term := os.Getenv("TERM")
	return os.Getenv("TERM_PROGRAM") == "ghostty" || os.Getenv("KITTY_WINDOW_ID") != "" ||
		strings.Contains(term, "kitty") || strings.Contains(term, "ghostty")
}

// kittyUpload devuelve las secuencias que transmiten el logo para cada tamaño.
func kittyUpload() string {
	data, err := kittyFS.ReadFile("assets/banner_kitty.png")
	if err != nil {
		return ""
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	var sb strings.Builder
	for _, d := range kittySizes {
		for i := 0; i < len(b64); i += 4096 {
			end := min(i+4096, len(b64))
			more := 1
			if end == len(b64) {
				more = 0
			}
			if i == 0 {
				fmt.Fprintf(&sb, "\x1b_Ga=T,f=100,U=1,q=2,i=%d,c=%d,r=%d,m=%d;%s\x1b\\", d.id, d.cols, d.rows, more, b64[i:end])
			} else {
				fmt.Fprintf(&sb, "\x1b_Gm=%d;%s\x1b\\", more, b64[i:end])
			}
		}
	}
	return sb.String()
}

// kittyArt dibuja los placeholders: cada celda lleva fila y columna como diacríticos.
func kittyArt(k bannerKind) string {
	d, ok := kittySizes[k]
	if !ok || d.cols > len(kittyDiacritics) || d.rows > len(kittyDiacritics) {
		return ""
	}
	var sb strings.Builder
	for r := 0; r < d.rows; r++ {
		if r > 0 {
			sb.WriteByte('\n')
		}
		fmt.Fprintf(&sb, "\x1b[38;5;%dm", d.id)
		for c := 0; c < d.cols; c++ {
			sb.WriteRune(0x10EEEE)
			sb.WriteRune(kittyDiacritics[r])
			sb.WriteRune(kittyDiacritics[c])
		}
		sb.WriteString("\x1b[39m")
	}
	return sb.String()
}

// kittyPad centra por columnas conocidas (no depende de medir los placeholders).
func kittyPad(k bannerKind, width int) string {
	art := kittyArt(k)
	if art == "" {
		return ""
	}
	pad := max(0, (width-kittySizes[k].cols)/2)
	sp := strings.Repeat(" ", pad)
	return sp + strings.ReplaceAll(art, "\n", "\n"+sp)
}

// uploadKitty transmite las imágenes a stdout; se llama una vez dentro de la pantalla alternativa.
func uploadKitty() {
	if kittyEnabled() {
		fmt.Fprint(os.Stdout, kittyUpload())
	}
}
