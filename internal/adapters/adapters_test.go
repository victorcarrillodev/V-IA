package adapters

import (
	"path/filepath"
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

func TestBlockIncludesMemoryProtocol(t *testing.T) {
	b, err := ODDBlock()
	if err != nil || len(b.Content) < 500 {
		t.Fatalf("%v", err)
	}
}
