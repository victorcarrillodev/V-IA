package adapters

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

// OpenCodeVerifiedMajor es la versión mayor con la que se probó el plugin y los skills.
const OpenCodeVerifiedMajor = 1

var verRe = regexp.MustCompile(`(\d+)\.\d+\.\d+`)

// ParseMajor extrae la versión mayor de la salida de `opencode --version` (ok=false si no hay).
func ParseMajor(out string) (int, bool) {
	m := verRe.FindStringSubmatch(out)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}

// OpenCodeMajor ejecuta `opencode --version` (máx. 15 s) y devuelve la versión mayor y el texto.
func OpenCodeMajor() (major int, raw string, ok bool) {
	path, err := exec.LookPath("opencode")
	if err != nil {
		return 0, "", false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return 0, "", false
	}
	raw = string(out)
	major, ok = ParseMajor(raw)
	return
}
