# Capacidades verificadas (Fase 0)

Fecha: 2026-11-23.

## Engram 3.7.2 (verificado en este PC)
- Binario único Go + SQLite/FTS5. Instalado en `~/.local/bin/engram` (release `linux_amd64`, checksum OK).
- MCP por stdio: `engram mcp [--tools=agent|admin|all] [--project NAME]` (`agent` = 23 herramientas).
- Integración con agentes: `engram setup <agent>` (claude-code, opencode, gemini-cli, codex, cursor, vscode-copilot, windsurf, …).
- CLI: `search`, `save`, `context`, `stats`, `sync` / `sync --import` (carpeta `.engram/`), `projects list`, `doctor`, `tui`, `export`/`import`.
- Cloud: opt-in (`engram cloud ...`), fuera de alcance.
- Decisión: V-IA **no** reimplementa memoria; delega en `engram setup claude-code` (o registra `engram mcp`) y solo aporta el protocolo (`memory-protocol.md`).

## Claude Code
- Instrucciones globales: `~/.claude/CLAUDE.md` (existe, con reglas del usuario → solo bloque gestionado).
- Skills nativas: `~/.claude/skills/`. Hooks y MCP: `~/.claude/settings.json`.
- Versión: no verificada (pendiente de `claude --version`).

## Agentes soportados (instrucciones globales)
Rutas tomadas de la configuración oficial de Gentle-AI (repo `internal/agents`, 22-nov-2026).

| Agente | Archivo | Cumplimiento |
|---|---|---|
| claude-code | `~/.claude/CLAUDE.md` + hooks + skills + subagente revisor | **bloqueo (hooks)**; verificado E1, E2, E3, E7 (`scripts/e2e-claude.sh`) |
| opencode | `$XDG_CONFIG_HOME/opencode/AGENTS.md` + `plugins/v-ia.js` + skills | **bloqueo (plugin)**; verificado con OpenCode 1.19.6: E1–E4 por instrucciones y E5 por bloqueo real de la herramienta `edit` (`scripts/e2e-opencode.sh`) |
| codex | `$CODEX_HOME/AGENTS.md` + `hooks.json` + skills | hooks **experimentales** (sin verificar con el agente real; exigen confianza en `/hooks`) |
| gemini-cli | `~/.gemini/GEMINI.md` + `settings.json` (hooks) + skills | hooks **experimentales** (sin verificar con el agente real) |
| cursor | `~/.cursor/hooks.json` (hooks) + skills | hooks **experimentales**; sin archivo global de instrucciones |
| kilocode | `$XDG_CONFIG_HOME/kilo/AGENTS.md` | solo instrucciones |
| qwen-code | `~/.qwen/QWEN.md` | solo instrucciones |
| windsurf | `~/.codeium/windsurf/memories/global_rules.md` | solo instrucciones |
| pi | `~/.pi/agent/APPEND_SYSTEM.md` | solo instrucciones; ruta tomada de gentle-ai, sin verificar con el agente |
| openclaw | `~/AGENTS.md` + skills en `~/.openclaw/skills` | solo instrucciones; ruta tomada de gentle-ai, sin verificar con el agente |
| copilot-cli | `$COPILOT_HOME/copilot-instructions.md` (`~/.copilot`) | solo instrucciones |
| generic | `AGENTS.md` del proyecto (`--agent generic`) | solo instrucciones |

## Pendiente de verificar
- Rutas de codex, gemini-cli, kilocode, qwen-code y windsurf: tomadas del código de Gentle-AI, no verificadas ejecutando cada agente (no están instalados aquí).
- Cursor, Copilot y el resto de agentes de Gentle-AI: no soportados todavía.
- Hooks de bloqueo en agentes distintos de Claude Code y OpenCode; los de Gemini CLI, Codex y Cursor están escritos según su documentación oficial (leída el 2026-11-24). Con las CLIs reales instaladas en una carpeta temporal (clave API falsa, HOME temporal; autorizado por el usuario) se verificó que el hook de petición de Gemini CLI y el de Codex 0.172.0 se ejecutan y clasifican la petición. Sin verificar con el agente real: los hooks previos a la herramienta de ambos (necesitan un modelo que responda) y todo Cursor.
- Windows: compila y `go vet` pasa para windows/amd64, pero no se ha ejecutado ni una prueba allí (el CI lo corre como experimental).
- macOS: compila para darwin/arm64 y darwin/amd64; las pruebas corren en CI, no localmente.
- Carpetas de skills de codex, gemini-cli, kilocode, qwen-code, windsurf y copilot-cli (las de claude-code y opencode se comprobaron con el agente real —OpenCode las lista con `opencode debug skill`—; el resto sigue la convención del agente sin ejecutarlo).
- Registro de Context7/Engram en OpenCode (`opencode.json`): la configuración escrita la valida `opencode debug config`; no se probó la conexión real con el servicio.
- Engram: no está instalado en este equipo; la integración (`install --engram`, hook SessionStart) está probada con un `engram` simulado.

## Comparación con gentle-ai (2026-11-24)

Método: lectura del código y docs de `~/Escritorio/gentle-ai` (v5.0.1, commit 459400e) y `inspect_go_binary` de REA sobre ambos binarios compilados.

| Aspecto | gentle-ai | V-IA |
|---|---|---|
| Dependencias | 26 (toml, yaml.v3, go-minisign, x/crypto, bubbles…) | 16, sin firma minisign: verifica `sha256` + atestación Sigstore (`gh`) |
| Agentes | 22+ (incl. pi, openclaw, conductor, kiro CLI) | 16 incl. pi y openclaw (rutas de gentle-ai, sin verificar con el agente). **conductor**: gentle-ai solo lo detecta (`~/.conductor`) y no escribe nada para él; V-IA tampoco, así que el resultado es equivalente |
| Skills | 11 propias + guía de estilo | 18; guía de estilo exigida por `core/skills_style_test.go` |
| Backups | 5 recientes + pin | igual: `restore --pin/--unpin`, poda automática a 5 |
| Canales | stable / beta / nightly | `update --channel stable\|beta` (`nightly` = alias de beta) |
| Alcance | `--scope global\|workspace` | `--scope global\|workspace` en install/sync/uninstall; workspace admite claude-code, gemini-cli, qwen-code, kimi y generic (escribe en el proyecto; backups y registro siguen en el HOME) |
| Review mode (on/off, global/clon) | sí | sí (`review mode`, `--scope global\|project`) |
| Store de revisiones | acumula; `review store-reset` | un único `.v-ia/receipt.json` que se sobrescribe: no acumula, no necesita reset |
| Continuidad / rutas de implementación | `work-continuity`, `trigger-rules` | cubierto en `core/odd/odd.md` y `classify.md` |
| Firma de releases | minisign + SignPath | atestación Sigstore + firma minisign de `checksums.txt` **implementada y probada**; se activa al configurar la clave (ver manual, `update`) |
| Telemetría | colector propio | `internal/telemetry` local |

**Versión de OpenCode:** `v-ia doctor` ejecuta `opencode --version` (máx. 15 s) y avisa si la versión mayor no es la verificada (1.x). gentle-ai va más lejos: tiene activos distintos para OpenCode V2 (2.0.4) y se niega a escribir con versión desconocida; V-IA solo avisa y no tiene plugin específico para V2.

**Detalles tomados de `agents.md` de gentle-ai:** Pi respeta `PI_CODING_AGENT_DIR` (`~/…` contra el home; absoluta solo en el HOME real); Kimi usa `~/.kimi-code` (`AGENTS.md`) y, solo si falta, la raíz heredada `~/.kimi` (`KIMI.md`, skills en `~/.config/agents/skills`). **Diferencias a propósito:** gentle-ai no escribe el prompt de Pi (lo gestiona su paquete Gentle Shell) y V-IA sí, en `APPEND_SYSTEM.md`, porque no tiene ese paquete; OpenClaw en gentle-ai usa el workspace de su configuración y V-IA escribe `~/AGENTS.md` (sin verificar). El identificador de Copilot CLI es `copilot-cli` en V-IA y `github-copilot-cli` en gentle-ai.
