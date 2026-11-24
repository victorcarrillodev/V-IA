// Package standards revisa los cambios en staging contra las reglas del proyecto
// (.v-ia/standards.md). Las reglas path y deny se evalúan en local, sin tokens;
// solo las reglas ai se delegan en un agente, en una única llamada por revisión.
package standards

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// RulesFile es la ruta relativa del archivo de reglas.
const RulesFile = ".v-ia/standards.md"

type Kind string

const (
	Path Kind = "path" // los archivos NUEVOS que casan con paths deben estar bajo where
	Deny Kind = "deny" // las líneas AÑADIDAS en paths no pueden casar con pattern
	AI   Kind = "ai"   // criterio en lenguaje natural (text), revisado por un agente
)

type Rule struct {
	ID      string
	Kind    Kind
	Paths   []string
	Where   []string
	Pattern *regexp.Regexp
	Text    string
	Message string
}

// Template es lo que escribe 'standards init'.
const Template = `# Estándares del proyecto (v-ia standards)
#
# Reglas separadas por una línea "---". Cada regla: clave: valor.
# Tipos:
#   path  los archivos NUEVOS que casan con paths deben estar bajo where   (sin tokens)
#   deny  las líneas AÑADIDAS en paths no pueden casar con pattern        (sin tokens)
#   ai    criterio en lenguaje natural en text; lo revisa un agente       (gasta tokens)
# paths/where aceptan globs separados por coma (** cruza carpetas).
# Borra los ejemplos y escribe los tuyos.

id: tests-junto-al-codigo
type: path
paths: **/*_test.go
where: internal/**, cmd/**
message: los tests van junto al código, en internal/ o cmd/

---
id: sin-println
type: deny
paths: internal/**/*.go
pattern: \bprintln\(
message: usa el logger del proyecto, no println

---
id: errores-con-contexto
type: ai
paths: internal/**/*.go
text: Los errores devueltos deben envolverse con contexto (fmt.Errorf con %w), no devolverse desnudos.
`

var idRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

func splitList(s string) []string {
	var out []string
	for _, x := range strings.Split(s, ",") {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}

// Parse lee las reglas. Una regla mal formada es un error (con su número), no se ignora.
func Parse(src string) ([]Rule, error) {
	var rules []Rule
	seen := map[string]bool{}
	for n, block := range splitBlocks(src) {
		kv := map[string]string{}
		for _, line := range strings.Split(block, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			k, v, ok := strings.Cut(line, ":")
			if !ok {
				return nil, fmt.Errorf("regla %d: línea sin 'clave: valor': %q", n+1, line)
			}
			kv[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
		if len(kv) == 0 {
			continue
		}
		r := Rule{ID: kv["id"], Kind: Kind(kv["type"]), Paths: splitList(kv["paths"]), Where: splitList(kv["where"]), Text: kv["text"], Message: kv["message"]}
		if !idRe.MatchString(r.ID) {
			return nil, fmt.Errorf("regla %d: id inválido %q (minúsculas, dígitos, . _ -)", n+1, r.ID)
		}
		if seen[r.ID] {
			return nil, fmt.Errorf("regla %q repetida", r.ID)
		}
		seen[r.ID] = true
		if len(r.Paths) == 0 {
			return nil, fmt.Errorf("regla %q: falta paths", r.ID)
		}
		switch r.Kind {
		case Path:
			if len(r.Where) == 0 {
				return nil, fmt.Errorf("regla %q: type path necesita where", r.ID)
			}
		case Deny:
			re, err := regexp.Compile(kv["pattern"])
			if kv["pattern"] == "" || err != nil {
				return nil, fmt.Errorf("regla %q: pattern inválido: %v", r.ID, err)
			}
			r.Pattern = re
		case AI:
			if r.Text == "" {
				return nil, fmt.Errorf("regla %q: type ai necesita text", r.ID)
			}
		default:
			return nil, fmt.Errorf("regla %q: type debe ser path, deny o ai (es %q)", r.ID, r.Kind)
		}
		rules = append(rules, r)
	}
	return rules, nil
}

func splitBlocks(src string) []string {
	var blocks []string
	var cur []string
	for _, l := range strings.Split(src, "\n") {
		if strings.TrimSpace(l) == "---" {
			blocks = append(blocks, strings.Join(cur, "\n"))
			cur = nil
			continue
		}
		cur = append(cur, l)
	}
	return append(blocks, strings.Join(cur, "\n"))
}

// Load lee las reglas del proyecto. found=false si no hay archivo.
func Load(root string) (rules []Rule, found bool, err error) {
	b, err := os.ReadFile(filepath.Join(root, RulesFile))
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	rules, err = Parse(string(b))
	return rules, true, err
}

// ---- globs: * dentro de una carpeta, ** entre carpetas, ? un carácter ----

func globRe(g string) *regexp.Regexp {
	var sb strings.Builder
	sb.WriteString("^")
	for i := 0; i < len(g); i++ {
		switch c := g[i]; c {
		case '*':
			if i+1 < len(g) && g[i+1] == '*' {
				i++
				if i+1 < len(g) && g[i+1] == '/' {
					i++
					sb.WriteString("(?:.*/)?") // **/ = cero o más carpetas
				} else {
					sb.WriteString(".*")
				}
			} else {
				sb.WriteString("[^/]*")
			}
		case '?':
			sb.WriteString("[^/]")
		default:
			sb.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	sb.WriteString("$")
	return regexp.MustCompile(sb.String())
}

// Match indica si path (con /) casa con algún glob.
func Match(globs []string, path string) bool {
	for _, g := range globs {
		if globRe(g).MatchString(path) {
			return true
		}
	}
	return false
}
