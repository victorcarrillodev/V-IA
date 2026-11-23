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
| opencode | `$XDG_CONFIG_HOME/opencode/AGENTS.md` | solo instrucciones (**verificado con OpenCode 1.19.5**: lee el archivo y cumple E1–E4; `scripts/e2e-opencode.sh`) |
| codex | `$CODEX_HOME/AGENTS.md` (`~/.codex`) | solo instrucciones |
| gemini-cli | `~/.gemini/GEMINI.md` | solo instrucciones |
| kilocode | `$XDG_CONFIG_HOME/kilo/AGENTS.md` | solo instrucciones |
| qwen-code | `~/.qwen/QWEN.md` | solo instrucciones |
| windsurf | `~/.codeium/windsurf/memories/global_rules.md` | solo instrucciones |
| generic | `AGENTS.md` del proyecto (`--agent generic`) | solo instrucciones |

## Pendiente de verificar
- Rutas de codex, gemini-cli, kilocode, qwen-code y windsurf: tomadas del código de Gentle-AI, no verificadas ejecutando cada agente (no están instalados aquí).
- Cursor, Copilot y el resto de agentes de Gentle-AI: no soportados todavía.
- Hooks de bloqueo en agentes distintos de Claude Code.
- Engram: no está instalado en este equipo; la integración (`install --engram`, hook SessionStart) está probada con un `engram` simulado.
