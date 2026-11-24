package standards

import (
	"bufio"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Added es una línea añadida con su número en el archivo nuevo.
type Added struct {
	N    int
	Text string
}

// File es un archivo cambiado: solo importan sus líneas añadidas.
type File struct {
	Path  string
	New   bool // archivo nuevo (status A)
	Added []Added
}

// Skipped son archivos que no se revisan: generados, lockfiles, binarios.
var skipRe = regexp.MustCompile(`(^|/)(go\.sum|package-lock\.json|yarn\.lock|pnpm-lock\.yaml|Cargo\.lock|poetry\.lock)$|\.(min\.js|min\.css|map|png|jpe?g|gif|ico|pdf|zip|gz|woff2?)$|(^|/)(vendor|node_modules|dist)/|\.pb\.go$|_generated\.go$`)

var hunkRe = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// ParseDiff interpreta 'git diff --unified=0'.
func ParseDiff(diff string) []File {
	var files []File
	var cur *File
	n := 0
	sc := bufio.NewScanner(strings.NewReader(diff))
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	for sc.Scan() {
		l := sc.Text()
		switch {
		case strings.HasPrefix(l, "diff --git "):
			files = append(files, File{})
			cur = &files[len(files)-1]
		case cur == nil:
		case strings.HasPrefix(l, "new file mode"):
			cur.New = true
		case strings.HasPrefix(l, "+++ b/"):
			cur.Path = strings.TrimPrefix(l, "+++ b/")
		case strings.HasPrefix(l, "+++ /dev/null"):
			cur.Path = "" // borrado: nada que revisar
		case strings.HasPrefix(l, "@@"):
			if m := hunkRe.FindStringSubmatch(l); m != nil {
				n, _ = strconv.Atoi(m[1])
			}
		case strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++"):
			cur.Added = append(cur.Added, Added{n, l[1:]})
			n++
		}
	}
	var out []File
	for _, f := range files {
		if f.Path != "" && !skipRe.MatchString(f.Path) {
			out = append(out, f)
		}
	}
	return out
}

// Source de los cambios a revisar.
type Source string

const (
	Staged Source = "staged" // git diff --cached (pre-commit)
	Head   Source = "head"   // git diff HEAD (árbol de trabajo)
)

func Changes(root string, src Source) ([]File, error) {
	args := []string{"-C", root, "diff", "--unified=0", "--no-color", "--no-ext-diff", "--diff-filter=ACMR"}
	if src == Staged {
		args = append(args, "--cached")
	} else {
		args = append(args, "HEAD")
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, err
	}
	return ParseDiff(string(out)), nil
}
