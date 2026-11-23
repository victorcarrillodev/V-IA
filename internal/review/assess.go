package review

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	TierPassive = "passive"
	TierMedium  = "medium"
	TierHigh    = "high"
	TierUnknown = "unknown" // una evaluación fallida nunca implica riesgo bajo
)

// Assessment es la evaluación de riesgo del cambio (lectura, no escribe nada).
type Assessment struct {
	Tier      string   `json:"tier"`
	Reasons   []string `json:"reasons"`
	Files     int      `json:"files"`
	Lines     int      `json:"lines"`
	ReviewDue bool     `json:"review_due"`
}

var (
	docRe      = regexp.MustCompile(`(?i)\.(md|txt|rst|adoc)$|(^|/)(docs?|LICENSE)(/|$)`)
	testRe     = regexp.MustCompile(`(?i)_test\.go$|\.(test|spec)\.[a-z]+$|(^|/)(tests?|__tests__|testdata)/`)
	highPathRe = regexp.MustCompile(`(?i)(^|/)(auth\w*|security|crypto\w*|secrets?|tokens?|passw\w*|permissions?|migrations?|schema\w*)(/|\.|_|-|$)|\.sql$|(^|/)\.github/workflows/|(^|/)Dockerfile|(^|/)(release|deploy)\w*(/|\.|$)`)
	depRe      = regexp.MustCompile(`(^|/)(go\.mod|go\.sum|package(-lock)?\.json|pnpm-lock\.yaml|yarn\.lock|Cargo\.(toml|lock)|requirements[\w.-]*\.txt|pyproject\.toml)$`)
	highTextRe = regexp.MustCompile(`(?i)\b(DROP\s+(TABLE|DATABASE)|DELETE\s+FROM|ALTER\s+TABLE|TRUNCATE)\b|(api[_-]?key|secret|passw(or)?d|private[_-]?key)\s*[:=]|sync\.(Mutex|RWMutex|WaitGroup)|\bgo\s+func\b|atomic\.|chmod\s+-R|exec\.Command`)
)

func git(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

// Assess clasifica el cambio respecto a base (por defecto HEAD) más los archivos sin seguimiento.
func Assess(root, base string) Assessment {
	if base == "" {
		base = "HEAD"
	}
	if base == "HEAD" {
		if _, err := git(root, "rev-parse", "--verify", "-q", "HEAD"); err != nil {
			base = "4b825dc642cb6eb9a060e54bf8d69288fbee4904" // árbol vacío: repo sin commits
		}
	}
	num, err := git(root, "diff", "--numstat", base)
	if err != nil {
		return Assessment{Tier: TierUnknown, Reasons: []string{"no se pudo evaluar: " + err.Error()}}
	}
	files := map[string]int{}
	for _, l := range strings.Split(strings.TrimSpace(num), "\n") {
		f := strings.Split(l, "\t")
		if len(f) != 3 {
			continue
		}
		a, _ := strconv.Atoi(f[0])
		d, _ := strconv.Atoi(f[1])
		files[f[2]] = a + d
	}
	if un, err := git(root, "ls-files", "--others", "--exclude-standard"); err == nil {
		for _, f := range strings.Split(strings.TrimSpace(un), "\n") {
			if f != "" && !strings.HasPrefix(f, ".v-ia/") {
				files[f] = 1
			}
		}
	}
	added, _ := git(root, "diff", "-U0", base)
	return classify(files, added)
}

func classify(files map[string]int, diff string) Assessment {
	a := Assessment{Files: len(files)}
	src, tests, high := 0, 0, false
	for f, n := range files {
		a.Lines += n
		switch {
		case testRe.MatchString(f):
			tests++
		case docRe.MatchString(f):
		default:
			src++
		}
		if highPathRe.MatchString(f) {
			high = true
			a.Reasons = append(a.Reasons, "ruta sensible: "+f)
		}
	}
	for _, l := range strings.Split(diff, "\n") {
		if strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++") && highTextRe.MatchString(l) {
			high = true
			a.Reasons = append(a.Reasons, "contenido sensible (datos/secretos/concurrencia/exec): "+strings.TrimSpace(trunc(l[1:], 60)))
			break
		}
	}
	switch {
	case len(files) == 0:
		a.Tier = TierPassive
		a.Reasons = append(a.Reasons, "sin cambios")
	case high:
		a.Tier = TierHigh
	case src == 0:
		a.Tier = TierPassive
		a.Reasons = append(a.Reasons, "solo docs/tests")
	case a.Lines >= 150 || src >= 5:
		a.Tier = TierMedium
		a.Reasons = append(a.Reasons, fmt.Sprintf("cambio amplio (%d líneas, %d archivos de código)", a.Lines, src))
	case tests == 0 && a.Lines > 30:
		a.Tier = TierMedium
		a.Reasons = append(a.Reasons, "código sin tests en el mismo cambio")
	default:
		a.Tier = TierPassive
		a.Reasons = append(a.Reasons, "cambio pequeño y acotado")
	}
	a.ReviewDue = a.Tier == TierHigh
	return a
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
