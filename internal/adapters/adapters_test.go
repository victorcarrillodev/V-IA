package adapters

import (
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
	}
	for n, p := range want {
		a, ok := Find(n)
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

func TestOnlyClaudeHasHooks(t *testing.T) {
	for _, a := range All {
		if a.Hooks != (a.Name == "claude-code") {
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
