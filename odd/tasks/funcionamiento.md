---
feature: funcionamiento
status: done
created: 2026-11-24
updated: 2026-11-24
authorized-by: usuario / 2026-11-24 / /goal: cubrir todo lo factible de funcionamiento
delivery: ask-on-risk
test-runner: go test -count=1 ./...
---
## Specs
- S1: quiero que cubras todos los aspectos que se pueden hacer, no me interesa de momento la distribucion sino el funcionamiento

## Tareas
- [x] T-1 (S1) route=inline commit=- Skills: go-testing, skill-improver, systemic-issue-triage, cognitive-doc-design, skill-registry (12 → 17)
- [x] T-2 (S1) route=inline commit=- Skills por agente (codex, gemini, kilo, qwen, windsurf, copilot-cli, opencode) y adaptador copilot-cli
- [x] T-3 (S1) route=inline commit=- Plugin de OpenCode que bloquea por mecanismo (chat.message + tool.execute.before)
- [x] T-4 (S1) route=inline commit=- Bash que escribe archivos bloqueado al instante en modo lectura
- [x] T-5 (S1) route=inline commit=- TDD estricto (v-ia tdd on|off|status) aplicado por el hook
- [x] T-6 (S1) route=inline commit=- Presets, --agent auto, registro de instalación (install.json) y doctor de archivos gestionados
- [x] T-7 (S1) route=inline commit=- MCP de Context7 y Engram también en OpenCode
- [x] T-8 (S1) route=inline commit=- task verify: evidencia contrastada con el recibo
- [x] T-9 (S1) route=inline commit=- Documentación: manual, README, capabilities y modelo de amenazas
- [x] T-10 (S1) route=inline commit=- Personas (--persona mentor|direct, bloque gestionado) y temas de la TUI (V_IA_THEME)
- [x] T-11 (S1) route=inline commit=- Recuento local opt-in (v-ia telemetry) y v-ia bench
- [x] T-12 (S1) route=inline commit=- Hooks de Gemini CLI, Codex y Cursor (experimentales) con fusionador genérico y normalización de eventos/herramientas
- [x] T-13 (S1) route=inline commit=- Portabilidad: compila para Windows/macOS, shell portable en review, CI con Windows experimental
- [x] T-14 (S1) route=inline commit=- Verificación real: handshake MCP con el servicio de Context7 (scripts/check-context7.sh, parte del recibo)
- [x] T-15 (S1) route=inline commit=- Verificación real: e2e de Claude Code con modelo real (consume tokens: evidencia manual, fuera del recibo)
- [x] T-17 (S1) route=inline commit=- Verificación con CLIs reales (Gemini y Codex): hooks de petición ejecutados; corrección de la salida JSON para Codex
- [x] T-16 (S1) route=inline commit=- Menores del revisor: writesFiles (find/xargs/env/intérpretes/sh -c, menos falsos positivos), instalación que no aborta por un agente ajeno roto, install.json

## Evidencia
| Tarea | Comando / comprobación | Resultado observado | Fecha |
|---|---|---|---|
| T-1 | go test -count=1 ./... (orphans_test) | ok: las 17 skills coinciden con skillNames y declaran su name; `opencode debug skill` lista odd-resume y skill-improver desde la carpeta instalada | 2026-11-24 |
| T-2 | go test -count=1 ./... (TestSkillsInstalledForEveryAgentThatSupportsThem) | ok: 6 agentes × 17 skills | 2026-11-24 |
| T-3 | go test ./... (TestOpenCodeGetsPluginAndSkills) + scripts/e2e-opencode.sh E3 E5 con OpenCode 1.19.6 real | E3: editó como se pidió; E5: la herramienta edit falló con el mensaje de V-IA y hello.go quedó intacto | 2026-11-24 |
| T-4 | go test -count=1 ./... (TestWritesFiles, TestReadModeBlocksBashWrites) | ok: sed -i, redirecciones, tee, mv, rm, git apply bloqueados; grep, git status, 2>&1 y /tmp permitidos | 2026-11-24 |
| T-5 | go test -count=1 ./... (TestTDDStrict, TestIsTestPath) | ok: bloquea código sin prueba; permite pruebas, docs y config; permite tras añadir a_test.go | 2026-11-24 |
| T-6 | go test -count=1 ./... (install_test, lifecycle_test) y ciclo install/doctor/sync/uninstall en HOME temporal | ok: 57 líneas de diff, idempotente, uninstall retira plugin y skills | 2026-11-24 |
| T-7 | go test ./... (TestAddOpenCodeMCPPreservesAndIsIdempotent) + v-ia deps --yes --only context7 en HOME temporal + opencode debug config | ok: opencode.json escrito y leído por OpenCode como MCP remoto; segunda ejecución "todo está instalado" | 2026-11-24 |
| T-8 | go test -count=1 ./... (TestVerifyEvidenceAgainstReceipt) | ok | 2026-11-24 |
| T-9 | go test -count=1 ./... (TestCadaComandoEstaDocumentado) | ok: todos los comandos aparecen en docs/manual.md | 2026-11-24 |
| T-10 | go test ./... (TestGeminiHooksAndPersonaRoundTrip) | ok: bloques odd y persona conviven en un archivo; uninstall retira ambos y borra el archivo que creamos | 2026-11-24 |
| T-11 | go test ./... (telemetry_test) + v-ia bench -n 100 | ok: apagado por defecto, cuenta y se borra; el hook decide en 8–28 µs (Edit con consulta a git: 2,6 ms de mediana), sin contar el arranque del proceso | 2026-11-24 |
| T-12 | go test ./... (TestHooksMergeStripRoundTripAllAgents, TestCodexAndCursorShapesAreNormalized, TestGeminiEventsAndToolsAreNormalized) + `v-ia hook` con cargas de Cursor/Codex/Gemini | ok: bloquea Write/Delete/apply_patch/replace en modo lectura, responde JSON de Cursor; NO ejecutado por los agentes reales (el intento con las CLIs reales fue denegado por el clasificador de permisos) | 2026-11-24 |
| T-13 | GOOS=windows GOARCH=amd64 go build -o /dev/null ./cmd/v-ia; GOOS=darwin GOARCH=arm64 go build -o /dev/null ./cmd/v-ia | compilan; las pruebas no se ejecutaron en esos sistemas | 2026-11-24 |
| T-14 | sh scripts/check-context7.sh | ✓ Context7 responde al handshake MCP (serverInfo Context7 4.7.3) | 2026-11-24 |
| T-15 | scripts/e2e-claude.sh con Claude real (V_IA_BIN=binario compilado) | E1, E2, E3 y E7 ✓ (exit 0). No está en checks.txt a propósito: gastaría tokens en cada `review check` | 2026-11-24 |
| T-16 | go test ./... (TestWritesFiles ampliado, TestBrokenAgentConfigDoesNotAbortInstall) | ok: find -delete, xargs rm, env rm, python -c, bash -c bloqueados; awk '$1>3', grep "a>b", git stash list permitidos; un settings.json ajeno inválido solo da aviso | 2026-11-24 |
| T-17 | gemini -p y codex exec --dangerously-bypass-hook-trust en HOME/CODEX_HOME temporales, clave API falsa (autorizado por el usuario) | Gemini: BeforeAgent ejecutó v-ia hook y .v-ia/state.json quedó en mode=read. Codex: UserPromptSubmit pasó de "Failed" (texto plano) a "Completed" (JSON) y state.json quedó en mode=read. Sin verificar: BeforeTool/PreToolUse (requieren modelo) y Cursor | 2026-11-24 |

## Log
- L1: quiero que cubras todos los aspectos que se pueden hacer, no me interesa de momento la distribucion sino el funcionamiento
- L2: Límites que quedan: hooks previos a la herramienta de Gemini/Codex y todo Cursor sin ejecutar con el agente real (los de petición de Gemini y Codex sí, tras autorización explícita del usuario); pruebas en Windows/macOS solo en CI; firma de releases es distribución. Corrección L3: Codex y Cursor SÍ tienen hooks (lo había descartado sin investigarlo).
