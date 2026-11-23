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

## Pendiente de verificar
- OpenCode, Codex, Gemini CLI, Cursor, Copilot: rutas y capacidades (Fase 6).
