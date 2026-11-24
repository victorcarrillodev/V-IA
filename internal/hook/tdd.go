package hook

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func tddPath(root string) string { return filepath.Join(root, ".v-ia", "tdd") }

// TDDEnabled indica si el proyecto exige prueba antes que código (TDD estricto).
func TDDEnabled(root string) bool {
	b, err := os.ReadFile(tddPath(root))
	return err == nil && strings.TrimSpace(string(b)) == "on"
}

// SetTDD activa o desactiva el TDD estricto del proyecto.
func SetTDD(root string, on bool) error {
	if err := os.MkdirAll(filepath.Dir(tddPath(root)), 0o755); err != nil {
		return err
	}
	v := "off\n"
	if on {
		v = "on\n"
	}
	return os.WriteFile(tddPath(root), []byte(v), 0o644)
}

var (
	testPathRe = regexp.MustCompile(`(?i)(_test\.go$|(^|[/._-])(test|spec)s?[/._-]|\.(test|spec)\.[a-z]+$|(^|/)(tests?|__tests__|spec)/|(^|/)test_[^/]+\.py$|_test\.py$|Test\.java$|Tests?\.cs$)`)
	sourceExt  = map[string]bool{".go": true, ".py": true, ".js": true, ".jsx": true, ".ts": true, ".tsx": true, ".mjs": true,
		".rs": true, ".java": true, ".kt": true, ".rb": true, ".php": true, ".c": true, ".h": true, ".cpp": true, ".cs": true, ".swift": true, ".sh": true}
)

// IsTestPath indica si la ruta es un archivo de pruebas.
func IsTestPath(p string) bool { return testPathRe.MatchString(filepath.ToSlash(p)) }

// IsSourcePath indica si la ruta es código de producción (no prueba, no documentación ni configuración).
func IsSourcePath(p string) bool {
	return sourceExt[strings.ToLower(filepath.Ext(p))] && !IsTestPath(p)
}

// testChanged indica si el árbol de trabajo ya tiene alguna prueba nueva o modificada.
// Sin git no se puede saber: devuelve (false, false) y el control no se aplica.
func testChanged(root string) (changed, known bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", root, "status", "--porcelain", "-uall")
	var out bytes.Buffer
	cmd.Stdout = &out
	if cmd.Run() != nil {
		return false, false
	}
	for _, l := range strings.Split(out.String(), "\n") {
		if len(l) > 3 && IsTestPath(strings.TrimSpace(l[3:])) {
			return true, true
		}
	}
	return false, true
}

// tddGate bloquea editar código de producción antes de existir una prueba que falle.
func tddGate(root, path string) string {
	if path == "" || !TDDEnabled(root) || !IsSourcePath(path) {
		return ""
	}
	if changed, known := testChanged(root); !known || changed {
		return ""
	}
	return "V-IA/TDD estricto: escribe primero una prueba que falle (añade o modifica un archivo de pruebas) y solo después edita " + filepath.Base(path) + ". Desactívalo con `v-ia tdd off`."
}
