package standards

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	hookBegin = "# >>> v-ia:standards >>>"
	hookEnd   = "# <<< v-ia:standards <<<"
)

func hookPath(root string) (string, error) {
	out, err := exec.Command("git", "-C", root, "rev-parse", "--git-path", "hooks/pre-commit").Output()
	if err != nil {
		return "", fmt.Errorf("no es un repositorio git: %w", err)
	}
	p := strings.TrimSpace(string(out))
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	return p, nil
}

func block(exe string, strict bool) string {
	cmd := fmt.Sprintf("%q standards check --hook", exe)
	if strict {
		cmd += " --strict"
	}
	return hookBegin + "\n[ -n \"$V_IA_SKIP_STANDARDS\" ] || " + cmd + " || exit 1\n" + hookEnd + "\n"
}

func strip(s string) (string, bool) {
	i, j := strings.Index(s, hookBegin), strings.Index(s, hookEnd)
	if i < 0 || j < i {
		return s, false
	}
	return s[:i] + strings.TrimPrefix(s[j+len(hookEnd):], "\n"), true
}

// InstallHook añade (o actualiza) el bloque en pre-commit conservando el hook existente.
func InstallHook(root, exe string, strict bool) (string, error) {
	p, err := hookPath(root)
	if err != nil {
		return "", err
	}
	old, _ := os.ReadFile(p)
	s := string(old)
	s, _ = strip(s)
	if s == "" {
		s = "#!/bin/sh\n"
	} else if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	s += block(exe, strict)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	return p, os.WriteFile(p, []byte(s), 0o755)
}

// RemoveHook quita el bloque; si el hook queda vacío (solo shebang) lo borra.
func RemoveHook(root string) (removed bool, err error) {
	p, err := hookPath(root)
	if err != nil {
		return false, err
	}
	old, err := os.ReadFile(p)
	if err != nil {
		return false, nil
	}
	s, ok := strip(string(old))
	if !ok {
		return false, nil
	}
	if strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "#!/bin/sh")) == "" {
		return true, os.Remove(p)
	}
	return true, os.WriteFile(p, []byte(s), 0o755)
}

// HookState informa si el hook está instalado y si es estricto.
func HookState(root string) (installed, strict bool) {
	p, err := hookPath(root)
	if err != nil {
		return false, false
	}
	b, err := os.ReadFile(p)
	if err != nil || !strings.Contains(string(b), hookBegin) {
		return false, false
	}
	return true, strings.Contains(string(b), "--hook --strict")
}
