package standards

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Runner envía un prompt a un agente y devuelve su respuesta. Sustituible en pruebas.
type Runner func(prompt string, timeout time.Duration) (string, error)

const (
	maxFileBytes  = 6000  // tope de diff por archivo enviado al agente
	maxTotalBytes = 24000 // tope de diff por llamada
	maxCache      = 2000
)

// ClaudeRunner usa 'claude -p' sin herramientas, hooks ni sesión: una sola respuesta barata.
func ClaudeRunner(model string) Runner {
	return func(prompt string, timeout time.Duration) (string, error) {
		if _, err := exec.LookPath("claude"); err != nil {
			return "", fmt.Errorf("claude no está en PATH")
		}
		if model == "" {
			model = "haiku"
		}
		cmd := exec.Command("claude", "-p", "--model", model, "--tools", "", "--no-session-persistence",
			"--disable-slash-commands", "--strict-mcp-config", "--setting-sources", "", "--output-format", "text")
		cmd.Stdin = strings.NewReader(prompt)
		var out, errb bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errb
		if err := cmd.Start(); err != nil {
			return "", err
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case err := <-done:
			if err != nil {
				return "", fmt.Errorf("claude: %v: %s", err, strings.TrimSpace(errb.String()))
			}
			return out.String(), nil
		case <-time.After(timeout):
			cmd.Process.Kill()
			return "", fmt.Errorf("claude no respondió en %s", timeout)
		}
	}
}

// Stats describe qué hizo la revisión con IA.
type Stats struct {
	Pairs, Cached, Sent, Skipped int
	Called                       bool
}

type cacheEntry struct {
	Findings []Finding `json:"findings"`
}

type cacheFile map[string]cacheEntry

func cachePath(root string) string { return filepath.Join(root, ".v-ia", "standards-cache.json") }

func loadCache(root string) cacheFile {
	c := cacheFile{}
	if b, err := os.ReadFile(cachePath(root)); err == nil {
		json.Unmarshal(b, &c)
	}
	return c
}

func saveCache(root string, c cacheFile) {
	if len(c) > maxCache {
		c = cacheFile{} // reinicio simple: la caché es solo una optimización
	}
	b, _ := json.Marshal(c)
	os.MkdirAll(filepath.Dir(cachePath(root)), 0o755)
	os.WriteFile(cachePath(root), b, 0o644)
}

// CacheSize devuelve cuántas revisiones hay guardadas.
func CacheSize(root string) int { return len(loadCache(root)) }

func hunkText(f File) string {
	var sb strings.Builder
	for _, a := range f.Added {
		fmt.Fprintf(&sb, "%d: %s\n", a.N, a.Text)
	}
	return sb.String()
}

func key(r Rule, f File, hunk string) string {
	h := sha256.Sum256([]byte(r.ID + "\x00" + r.Text + "\x00" + f.Path + "\x00" + hunk))
	return hex.EncodeToString(h[:])
}

type pair struct {
	rule Rule
	file File
	hunk string
	key  string
}

// ReviewAI revisa las reglas ai: caché por (regla, archivo, fragmento), una sola llamada
// para todo lo pendiente. Un error de infraestructura se devuelve aparte (fail-open).
func ReviewAI(root string, rules []Rule, files []File, run Runner, timeout time.Duration) ([]Finding, Stats, error) {
	var st Stats
	cache := loadCache(root)
	var found []Finding
	var pending []pair
	for _, r := range rules {
		if r.Kind != AI {
			continue
		}
		for _, f := range files {
			if !Match(r.Paths, f.Path) || len(f.Added) == 0 {
				continue
			}
			st.Pairs++
			h := hunkText(f)
			k := key(r, f, h)
			if e, ok := cache[k]; ok {
				st.Cached++
				found = append(found, e.Findings...)
				continue
			}
			pending = append(pending, pair{r, f, h, k})
		}
	}
	if len(pending) == 0 {
		sortFindings(found)
		return found, st, nil
	}
	// presupuesto de tamaño: lo que no cabe se omite (y se dice)
	var sel []pair
	total := 0
	for _, p := range pending {
		if len(p.hunk) > maxFileBytes {
			p.hunk = p.hunk[:maxFileBytes] + "…(recortado)\n"
		}
		if total+len(p.hunk) > maxTotalBytes {
			st.Skipped++
			continue
		}
		total += len(p.hunk)
		sel = append(sel, p)
	}
	st.Sent, st.Called = len(sel), true
	out, err := run(buildPrompt(sel), timeout)
	if err != nil {
		sortFindings(found)
		return found, st, err
	}
	res, err := parseFindings(out)
	if err != nil {
		sortFindings(found)
		return found, st, err
	}
	for _, p := range sel {
		var mine []Finding
		for _, f := range res {
			if f.Rule == p.rule.ID && f.File == p.file.Path {
				mine = append(mine, f)
			}
		}
		cache[p.key] = cacheEntry{mine}
		found = append(found, mine...)
	}
	saveCache(root, cache)
	sortFindings(found)
	return found, st, nil
}

func buildPrompt(sel []pair) string {
	var sb strings.Builder
	sb.WriteString("Eres un revisor de estándares de código. Revisa SOLO las líneas añadidas (con su número) contra cada regla y señala únicamente incumplimientos claros; ante la duda, no los señales.\n\nREGLAS:\n")
	seen := map[string]bool{}
	for _, p := range sel {
		if !seen[p.rule.ID] {
			seen[p.rule.ID] = true
			fmt.Fprintf(&sb, "- %s: %s\n", p.rule.ID, p.rule.Text)
		}
	}
	sb.WriteString("\nARCHIVOS (cada uno indica qué reglas le aplican):\n")
	for _, p := range sel {
		fmt.Fprintf(&sb, "\n### archivo: %s  regla: %s\n%s", p.file.Path, p.rule.ID, p.hunk)
	}
	sb.WriteString("\nResponde SOLO con un array JSON, sin texto alrededor ni markdown: [{\"rule\":\"id\",\"file\":\"ruta\",\"line\":N,\"msg\":\"motivo breve\"}]. Si no hay incumplimientos, responde [].")
	return sb.String()
}

func parseFindings(out string) ([]Finding, error) {
	i, j := strings.Index(out, "["), strings.LastIndex(out, "]")
	if i < 0 || j < i {
		return nil, fmt.Errorf("respuesta sin JSON: %.80q", out)
	}
	var raw []struct {
		Rule string `json:"rule"`
		File string `json:"file"`
		Line int    `json:"line"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal([]byte(out[i:j+1]), &raw); err != nil {
		return nil, fmt.Errorf("JSON inválido: %v", err)
	}
	var fs []Finding
	for _, r := range raw {
		fs = append(fs, Finding{Rule: r.Rule, File: r.File, Line: r.Line, Msg: r.Msg})
	}
	return fs, nil
}
