package standards

import (
	"fmt"
	"sort"
	"strings"
)

// Finding es un incumplimiento.
type Finding struct {
	Rule, File string
	Line       int // 0 si no aplica
	Msg        string
}

func (f Finding) String() string {
	loc := f.File
	if f.Line > 0 {
		loc = fmt.Sprintf("%s:%d", f.File, f.Line)
	}
	return fmt.Sprintf("%s  %s  %s", f.Rule, loc, f.Msg)
}

// Deterministic evalúa las reglas path y deny. No gasta tokens.
func Deterministic(rules []Rule, files []File) []Finding {
	var out []Finding
	for _, r := range rules {
		for _, f := range files {
			if !Match(r.Paths, f.Path) {
				continue
			}
			switch r.Kind {
			case Path:
				if f.New && !Match(r.Where, f.Path) {
					out = append(out, Finding{r.ID, f.Path, 0, orDefault(r.Message, "ubicación no permitida; debe estar en "+strings.Join(r.Where, ", "))})
				}
			case Deny:
				for _, a := range f.Added {
					if r.Pattern.MatchString(a.Text) {
						out = append(out, Finding{r.ID, f.Path, a.N, orDefault(r.Message, "patrón prohibido: "+r.Pattern.String())})
					}
				}
			}
		}
	}
	sortFindings(out)
	return out
}

func orDefault(s, d string) string {
	if s != "" {
		return s
	}
	return d
}

func sortFindings(f []Finding) {
	sort.SliceStable(f, func(i, j int) bool {
		if f[i].File != f[j].File {
			return f[i].File < f[j].File
		}
		return f[i].Line < f[j].Line
	})
}

// HasAI indica si alguna regla necesita agente.
func HasAI(rules []Rule) bool {
	for _, r := range rules {
		if r.Kind == AI {
			return true
		}
	}
	return false
}
