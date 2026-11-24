package corefs

import (
	"io/fs"
	"path"
	"regexp"
	"strings"
	"testing"
)

// Guía de estilo de skills (equivalente a la de gentle-ai): frontmatter completo,
// descripción de una línea ≤250 caracteres, y cuerpo acotado.
func TestSkillsStyle(t *testing.T) {
	ents, err := fs.ReadDir(FS, "skills")
	if err != nil {
		t.Fatal(err)
	}
	ver := regexp.MustCompile(`^\s+version:\s*"?\d+(\.\d+)*"?\s*$`)
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		b, err := fs.ReadFile(FS, path.Join("skills", e.Name(), "SKILL.md"))
		if err != nil {
			t.Errorf("%s: falta SKILL.md", e.Name())
			continue
		}
		lines := strings.Split(string(b), "\n")
		if lines[0] != "---" {
			t.Errorf("%s: sin frontmatter", e.Name())
			continue
		}
		end := 0
		for i := 1; i < len(lines); i++ {
			if lines[i] == "---" {
				end = i
				break
			}
		}
		if end == 0 {
			t.Errorf("%s: frontmatter sin cerrar", e.Name())
			continue
		}
		fm := strings.Join(lines[1:end], "\n")
		for _, k := range []string{"name: " + e.Name(), "description:", "license:", "metadata:", "author:"} {
			if !strings.Contains(fm, k) {
				t.Errorf("%s: frontmatter sin %q", e.Name(), k)
			}
		}
		hasVer := false
		for _, l := range lines[1:end] {
			if ver.MatchString(l) {
				hasVer = true
			}
			if v, ok := strings.CutPrefix(l, "description:"); ok && len(strings.TrimSpace(v)) > 250 {
				t.Errorf("%s: description de %d caracteres (máx. 250)", e.Name(), len(strings.TrimSpace(v)))
			}
		}
		if !hasVer {
			t.Errorf("%s: falta metadata.version", e.Name())
		}
		if w := len(strings.Fields(strings.Join(lines[end+1:], " "))); w > 700 {
			t.Errorf("%s: cuerpo de %d palabras (máx. 700)", e.Name(), w)
		}
	}
}
