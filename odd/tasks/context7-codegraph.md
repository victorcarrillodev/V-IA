---
feature: context7-codegraph
status: done
created: 2026-11-23
updated: 2026-11-23
authorized-by: <usuario / fecha / alcance>
delivery: ask-on-risk
test-runner: <comando>
---
## Specs
- S1: Agregar Context7 y CodeGraph como componentes opcionales (flags --context7 --codegraph --all, casillas en la TUI, doctor, docs y README). El GGA propio (v-ia standards) queda para después.

## Tareas
- [x] T-1 (S1) route=inline commit=- Paquete internal/extras (plan, ejecución, detección) con pruebas
- [x] T-2 (S1) route=inline commit=- Flags --context7 --codegraph --all en install y doctor
- [x] T-3 (S1) route=inline commit=- Casillas en la TUI y revisión con comandos
- [x] T-4 (S1) route=inline commit=- README y manual
- [x] T-5 (S1) route=inline commit=- Hallazgos del revisor RDD: checksum en install.sh, detección por mcpServers, dry-run sin fallar, extras antes de Apply, versiones y SHAs fijados

## Evidencia
| Tarea | Comando / comprobación | Resultado observado | Fecha |
|---|---|---|---|
| T-1 | go test -count=1 ./internal/extras | ok (7 pruebas: plan, idempotencia, requisitos, parada en error) | 2026-11-23 |
| T-2 | HOME=tmp v-ia install --dry-run --all | falla antes de escribir: "faltan requisitos: engram...; context7 necesita 'npx'...; codegraph necesita 'npm'" | 2026-11-23 |
| T-3 | go test -count=1 ./internal/tui | ok, incluye vista dentro de 60x18 y 2 pruebas nuevas de extras | 2026-11-23 |
| T-5 | go test -count=1 ./...; install.sh con release simulada | ok; hash correcto instala, hash manipulado aborta sin instalar; dry-run --all sin requisitos lista comandos y avisa | 2026-11-23 |
| T-4 | revisión manual de README y docs/manual.md | secciones 4.4 y 'Componentes opcionales' añadidas; sin ejecutar contra las herramientas reales | 2026-11-23 |

## Log
- L1: Agregar Context7 y CodeGraph como componentes opcionales (flags --context7 --codegraph --all, casillas en la TUI, doctor, docs y README). El GGA propio (v-ia standards) queda para después.
