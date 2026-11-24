package adapters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathsPerAgent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("CODEX_HOME", "")
	h := "/home/u"
	want := map[string]string{
		"claude-code": "/home/u/.claude/CLAUDE.md",
		"opencode":    "/home/u/.config/opencode/AGENTS.md",
		"codex":       "/home/u/.codex/AGENTS.md",
		"gemini-cli":  "/home/u/.gemini/GEMINI.md",
		"kilocode":    "/home/u/.config/kilo/AGENTS.md",
		"qwen-code":   "/home/u/.qwen/QWEN.md",
		"windsurf":    "/home/u/.codeium/windsurf/memories/global_rules.md",
		"copilot-cli": "/home/u/.copilot/copilot-instructions.md",
	}
	for _, n := range []string{"cursor", "kiro-ide", "vscode-copilot", "antigravity"} {
		want[n] = "" // cargan un archivo propio con frontmatter (OwnFile), no un bloque
	}
	want["trae-ide"] = "/home/u/.config/Trae/User/user_rules.md"
	want["hermes"] = "/home/u/.hermes/SOUL.md"
	want["kimi"] = "/home/u/.kimi-code/AGENTS.md"
	want["pi"] = "/home/u/.pi/agent/APPEND_SYSTEM.md"
	want["openclaw"] = "/home/u/AGENTS.md"
	for n, p := range want {
		a, ok := Find(n)
		if p == "" {
			if !ok || len(a.Targets(h)) != 0 || a.OwnFile == nil {
				t.Errorf("%s debe usar un archivo propio, no un bloque", n)
			}
			continue
		}
		if !ok || a.Targets(h)[0] != p {
			t.Errorf("%s: %v quería %s", n, a.Targets(h), p)
		}
	}
	if len(All) != len(want) {
		t.Errorf("hay %d adaptadores y %d casos", len(All), len(want))
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	t.Setenv("CODEX_HOME", "/cx")
	oc, _ := Find("opencode")
	cx, _ := Find("codex")
	if oc.Targets("/h")[0] != filepath.Join("/xdg", "opencode", "AGENTS.md") || cx.Targets("/h")[0] != "/cx/AGENTS.md" {
		t.Fatal("XDG_CONFIG_HOME / CODEX_HOME ignorados")
	}
}

func TestOwnFileAgentsCarryFrontmatterAndFitLimits(t *testing.T) {
	for _, a := range All {
		if a.OwnFile == nil {
			continue
		}
		p, fm := a.OwnFile("/home/u")
		if p == "" || !strings.HasPrefix(fm, "---\n") || !strings.HasSuffix(fm, "---\n") {
			t.Errorf("%s: frontmatter inválido %q", a.Name, fm)
		}
		b, err := ODDBlock(a.Hooks)
		if err != nil {
			t.Fatal(err)
		}
		if a.Name == "antigravity" && len(fm)+len(b.Content) > 24000 {
			t.Errorf("antigravity trunca los archivos de reglas por encima de 24000 bytes: %d", len(fm)+len(b.Content))
		}
	}
}

func TestOnlyHookCapableAgentsHaveHooks(t *testing.T) {
	for _, a := range All {
		if a.Hooks != (a.Name == "claude-code" || a.Name == "opencode" || a.Name == "gemini-cli" || a.Name == "codex" || a.Name == "cursor") {
			t.Errorf("%s: Hooks=%v", a.Name, a.Hooks)
		}
	}
}

func TestBlockMatchesAgentCapabilities(t *testing.T) {
	enf, err := ODDBlock(true)
	if err != nil {
		t.Fatal(err)
	}
	adv, err := ODDBlock(false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(enf.Content, "BLOQUEAN") || !strings.Contains(enf.Content, "odd-resume") {
		t.Fatal("el bloque con hooks debe describir bloqueos y skills")
	}
	for _, banned := range []string{"BLOQUEAN", "hook", "Skills disponibles", "odd-resume", "v-ia mode"} {
		if strings.Contains(adv.Content, banned) {
			t.Errorf("el bloque sin hooks no debe mencionar %q (no lo tiene)", banned)
		}
	}
	for _, must := range []string{"NADA te bloquea", "v-ia guard check", "v-ia task check", "v-ia review check", "gana el código"} {
		if !strings.Contains(adv.Content, must) {
			t.Errorf("el bloque sin hooks debe incluir %q", must)
		}
	}
	if enf.Content == adv.Content {
		t.Fatal("deben ser distintos")
	}
}

func TestSupportsWorkspace(t *testing.T) {
	for _, n := range WorkspaceAgents {
		if n != "generic" {
			if _, ok := Find(n); !ok {
				t.Errorf("%s en WorkspaceAgents pero no es un adaptador", n)
			}
		}
		if !SupportsWorkspace(n) {
			t.Errorf("%s debería admitir workspace", n)
		}
	}
	if SupportsWorkspace("codex") || SupportsWorkspace("opencode") {
		t.Error("codex/opencode no usan ~/.carpeta y no deben admitir workspace")
	}
}

func TestPiYKimiRutas(t *testing.T) {
	h := t.TempDir()
	t.Setenv("PI_CODING_AGENT_DIR", "")
	if got := piAgentDir(h); got != filepath.Join(h, ".pi", "agent") {
		t.Errorf("pi por defecto: %s", got)
	}
	t.Setenv("PI_CODING_AGENT_DIR", "~/aislado/agent")
	if got := piAgentDir(h); got != filepath.Join(h, "aislado", "agent") {
		t.Errorf("pi con ~: %s", got)
	}
	t.Setenv("PI_CODING_AGENT_DIR", "/etc/fuera")
	if got := piAgentDir(h); got != filepath.Join(h, ".pi", "agent") {
		t.Errorf("una ruta absoluta no puede sacar un home ajeno de sí mismo: %s", got)
	}
	// kimi: sin ninguna raíz → actual; solo ~/.kimi → heredada; ambas → actual
	if kimiLegacy(h) {
		t.Error("sin carpetas debe usar la disposición actual")
	}
	os.MkdirAll(filepath.Join(h, ".kimi"), 0o755)
	if !kimiLegacy(h) {
		t.Error("solo ~/.kimi debe usar la heredada")
	}
	os.MkdirAll(filepath.Join(h, ".kimi-code"), 0o755)
	if kimiLegacy(h) {
		t.Error("con ~/.kimi-code gana la actual")
	}
}
