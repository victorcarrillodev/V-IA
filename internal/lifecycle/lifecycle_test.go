package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"v-ia/internal/adapters"
	"v-ia/internal/core"
)

func opts(t *testing.T, agents ...string) Options {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	t.Setenv("CODEX_HOME", filepath.Join(home, "codex"))
	return Options{Home: home, Agents: agents, Exe: "/usr/bin/v-ia", Hooks: true, Permissions: true}
}

func apply(t *testing.T, p core.Plan, o Options) {
	t.Helper()
	if _, err := core.Apply(p, filepath.Join(o.Home, ".v-ia", "backups"), time.Now()); err != nil {
		t.Fatal(err)
	}
}

func mustPlan(t *testing.T, f func(Options) (core.Plan, error), o Options) core.Plan {
	t.Helper()
	p, err := f(o)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestInstallIsIdempotentAndComplete(t *testing.T) {
	o := opts(t, "claude-code", "codex")
	apply(t, mustPlan(t, Install, o), o)
	for _, f := range []string{
		".claude/CLAUDE.md", ".claude/settings.json", ".claude/agents/v-ia-reviewer.md",
		".claude/skills/odd-resume/SKILL.md", ".claude/skills/rdd-review/SKILL.md", "codex/AGENTS.md",
	} {
		if _, err := os.Stat(filepath.Join(o.Home, f)); err != nil {
			t.Errorf("falta %s", f)
		}
	}
	if n := len(mustPlan(t, Install, o).Changes()); n != 0 {
		t.Fatalf("la segunda instalación debe ser no-op, hay %d cambios", n)
	}
}

func TestSyncNeverAddsNewComponents(t *testing.T) {
	o := opts(t, "claude-code")
	if n := len(mustPlan(t, Sync, o).Changes()); n != 0 {
		t.Fatalf("sync sin instalación previa no debe crear nada: %d", n)
	}
	apply(t, mustPlan(t, Install, o), o)
	sk := filepath.Join(o.Home, ".claude", "skills", "odd-verify", "SKILL.md")
	os.Remove(sk)
	for _, a := range mustPlan(t, Sync, o).Changes() {
		if a.Path == sk {
			t.Fatal("sync no debe recrear lo que el usuario borró")
		}
	}
}

func TestUserEditedSkillIsKeptAndConflicts(t *testing.T) {
	o := opts(t, "claude-code")
	apply(t, mustPlan(t, Install, o), o)
	sk := filepath.Join(o.Home, ".claude", "skills", "odd-resume", "SKILL.md")
	b, _ := os.ReadFile(sk)
	os.WriteFile(sk, []byte(strings.Replace(string(b), "Retomar", "RETOMAR-MIO", 1)), 0o644)
	p := mustPlan(t, Install, o)
	if len(p.Conflicts()) != 1 || p.Conflicts()[0].Path != sk {
		t.Fatalf("debe haber 1 conflicto en la skill editada: %+v", p.Conflicts())
	}
	if _, err := core.Apply(p, filepath.Join(o.Home, "bk"), time.Now()); err == nil {
		t.Fatal("Apply debe negarse con conflictos")
	}
	for _, a := range mustPlan(t, Uninstall, o).Actions {
		if a.Path == sk {
			t.Fatal("uninstall no debe tocar una skill editada a mano")
		}
	}
}

func TestUninstallThenRestoreRoundTrip(t *testing.T) {
	o := opts(t, "claude-code")
	os.MkdirAll(filepath.Join(o.Home, ".claude"), 0o755)
	user := "# mis reglas\n"
	os.WriteFile(filepath.Join(o.Home, ".claude", "CLAUDE.md"), []byte(user), 0o644)
	os.WriteFile(adapters.SettingsPath(o.Home), []byte(`{"model":"opus"}`), 0o644)
	apply(t, mustPlan(t, Install, o), o)
	apply(t, mustPlan(t, Uninstall, o), o)
	if b, _ := os.ReadFile(filepath.Join(o.Home, ".claude", "CLAUDE.md")); string(b) != user {
		t.Fatalf("uninstall debe dejar solo lo del usuario: %q", b)
	}
	if _, err := os.Stat(filepath.Join(o.Home, ".claude", "skills", "odd-resume", "SKILL.md")); err == nil {
		t.Fatal("la skill debía borrarse")
	}
	if b, _ := os.ReadFile(adapters.SettingsPath(o.Home)); strings.Contains(string(b), "v-ia hook") || !strings.Contains(string(b), "opus") {
		t.Fatalf("settings tras uninstall: %s", b)
	}
	bk, _ := core.Latest(filepath.Join(o.Home, ".v-ia", "backups"))
	if err := core.Restore(bk); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(o.Home, ".claude", "skills", "odd-resume", "SKILL.md")); err != nil {
		t.Fatal("restore debe devolver la skill borrada")
	}
}

func TestUninstallDeletesFileWeCreated(t *testing.T) {
	o := opts(t, "codex")
	apply(t, mustPlan(t, Install, o), o)
	apply(t, mustPlan(t, Uninstall, o), o)
	if _, err := os.Stat(filepath.Join(o.Home, "codex", "AGENTS.md")); err == nil {
		t.Fatal("un AGENTS.md que solo contenía nuestro bloque debe borrarse")
	}
}

func TestNoHooksFlag(t *testing.T) {
	o := opts(t, "claude-code")
	o.Hooks, o.Permissions = false, false
	for _, a := range mustPlan(t, Install, o).Actions {
		if a.Path == adapters.SettingsPath(o.Home) {
			t.Fatal("sin hooks ni permisos no se toca settings.json")
		}
	}
}

func TestNoHooksButPermissionsDoesNotInstallHooks(t *testing.T) {
	o := opts(t, "claude-code")
	o.Hooks = false
	apply(t, mustPlan(t, Install, o), o)
	b, err := os.ReadFile(adapters.SettingsPath(o.Home))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "v-ia hook") || !strings.Contains(string(b), "permissions") {
		t.Fatalf("--no-hooks debe sembrar solo la deny-list:\n%s", b)
	}
}

func TestOpenCodeGetsPluginAndSkills(t *testing.T) {
	o := opts(t, "opencode")
	apply(t, mustPlan(t, Install, o), o)
	xdg := os.Getenv("XDG_CONFIG_HOME")
	plug, err := os.ReadFile(filepath.Join(xdg, "opencode", "plugins", "v-ia.js"))
	if err != nil {
		t.Fatalf("falta el plugin: %v", err)
	}
	if !strings.Contains(string(plug), `"/usr/bin/v-ia"`) || strings.Contains(string(plug), "__V_IA_EXE__") {
		t.Fatal("el plugin no recibió la ruta del binario")
	}
	if !strings.Contains(string(plug), "// v-ia:managed-file sha256=") {
		t.Fatal("el plugin debe llevar una marca de comentario JS, no HTML")
	}
	for _, n := range skillNames {
		if _, err := os.Stat(filepath.Join(xdg, "opencode", "skills", n, "SKILL.md")); err != nil {
			t.Errorf("falta la skill %s en OpenCode: %v", n, err)
		}
	}
	// idempotente y desinstalable sin dejar nada
	if p := mustPlan(t, Install, o); len(p.Changes()) != 0 {
		t.Fatalf("segunda instalación debía ser nula: %d cambios", len(p.Changes()))
	}
	apply(t, mustPlan(t, Uninstall, o), o)
	if _, err := os.Stat(filepath.Join(xdg, "opencode", "plugins", "v-ia.js")); err == nil {
		t.Fatal("uninstall debía retirar el plugin")
	}
}

func TestNoHooksSkipsPluginAndNoSkillsSkipsSkills(t *testing.T) {
	o := opts(t, "opencode")
	o.Hooks, o.NoSkills = false, true
	apply(t, mustPlan(t, Install, o), o)
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if _, err := os.Stat(filepath.Join(xdg, "opencode", "plugins", "v-ia.js")); err == nil {
		t.Fatal("sin hooks no hay plugin")
	}
	if _, err := os.Stat(filepath.Join(xdg, "opencode", "skills")); err == nil {
		t.Fatal("NoSkills no debe instalar skills")
	}
}

func TestSkillsInstalledForEveryAgentThatSupportsThem(t *testing.T) {
	o := opts(t, "codex", "gemini-cli", "kilocode", "qwen-code", "windsurf", "copilot-cli")
	p := mustPlan(t, Install, o)
	n := 0
	for _, a := range p.Actions {
		if strings.HasSuffix(a.Path, "SKILL.md") {
			n++
		}
	}
	if n != 6*len(skillNames) {
		t.Fatalf("esperaba %d skills, hay %d", 6*len(skillNames), n)
	}
}

func TestGeminiHooksAndPersonaRoundTrip(t *testing.T) {
	o := opts(t, "gemini-cli")
	o.Persona = "mentor"
	if err := os.MkdirAll(filepath.Join(o.Home, ".gemini"), 0o755); err != nil {
		t.Fatal(err)
	}
	pre := `{"theme":"dark"}` + "\n"
	sp := filepath.Join(o.Home, ".gemini", "settings.json")
	if err := os.WriteFile(sp, []byte(pre), 0o644); err != nil {
		t.Fatal(err)
	}
	apply(t, mustPlan(t, Install, o), o)
	b, _ := os.ReadFile(sp)
	if !strings.Contains(string(b), `"BeforeTool"`) || !strings.Contains(string(b), `"theme"`) {
		t.Fatalf("settings de Gemini: %s", b)
	}
	md, _ := os.ReadFile(filepath.Join(o.Home, ".gemini", "GEMINI.md"))
	if !strings.Contains(string(md), "id=odd") || !strings.Contains(string(md), "id=persona") {
		t.Fatalf("GEMINI.md debe llevar los bloques odd y persona:\n%s", md)
	}
	if p := mustPlan(t, Install, o); len(p.Changes()) != 0 {
		t.Fatalf("segunda instalación debía ser nula: %d cambios", len(p.Changes()))
	}
	apply(t, mustPlan(t, Uninstall, o), o)
	if b, _ := os.ReadFile(sp); strings.Contains(string(b), "BeforeTool") || !strings.Contains(string(b), `"theme"`) {
		t.Fatalf("uninstall debe quitar solo los hooks: %s", b)
	}
	if _, err := os.Stat(filepath.Join(o.Home, ".gemini", "GEMINI.md")); err == nil {
		t.Fatal("uninstall debía borrar GEMINI.md que creamos (odd + persona)")
	}
}

func TestBrokenAgentConfigDoesNotAbortInstall(t *testing.T) {
	o := opts(t, "claude-code", "gemini-cli")
	_ = os.MkdirAll(filepath.Join(o.Home, ".gemini"), 0o755)
	bad := filepath.Join(o.Home, ".gemini", "settings.json")
	_ = os.WriteFile(bad, []byte("{ // comentario\n}"), 0o644)
	p := mustPlan(t, Install, o)
	if len(p.Warnings) != 1 {
		t.Fatalf("debía avisar de la configuración inválida de Gemini: %v", p.Warnings)
	}
	apply(t, p, o)
	if b, _ := os.ReadFile(bad); string(b) != "{ // comentario\n}" {
		t.Fatal("no debe tocar la configuración inválida")
	}
	if _, err := os.Stat(filepath.Join(o.Home, ".claude", "settings.json")); err != nil {
		t.Fatal("el resto de agentes debía instalarse")
	}
}
