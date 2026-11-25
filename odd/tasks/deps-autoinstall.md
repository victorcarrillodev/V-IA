---
feature: deps-autoinstall
status: done
created: 2026-11-23
updated: 2026-11-23
authorized-by: <usuario / fecha / alcance>
delivery: ask-on-risk
test-runner: <comando>
---
## Specs
- S1: Que 'v-ia install --all' y un nuevo 'v-ia deps' instalen lo que falta (engram, codegraph, context7) en vez de fallar: Context7 por MCP remoto sin Node, Engram con go install/brew, CodeGraph con su instalador; nunca sudo; TUI muestra los pasos de instalación en la revisión.

## Tareas
- [x] T-1 (S1) route=inline commit=- extras.Plan instala lo que falta (Engram, CodeGraph) y Context7 por MCP remoto; solo falla si no hay forma de instalar
- [x] T-2 (S1) route=inline commit=- Comando v-ia deps [--yes] [--only]
- [x] T-3 (S1) route=inline commit=- Detección de Engram y doctor unificado
- [x] T-4 (S1) route=inline commit=- README y manual

## Evidencia
| Tarea | Comando / comprobación | Resultado observado | Fecha |
|---|---|---|---|
| T-1 | go test -count=1 ./internal/extras | ok: instalación por go/brew/npm/curl, idempotencia, solo falla sin claude o sin go/brew | 2026-11-23 |
| T-2 | HOME=tmp v-ia deps --yes --only engram,context7 (real) | engram v3.7.2 instalado en ~/.local/bin y registrado; context7 registrado en mcpServers; "listo" | 2026-11-23 |
| T-3 | go test -count=1 ./... | todos ok; Engram ya configurado se detecta y no se repite | 2026-11-23 |
| T-4 | revisión de README y manual | secciones 4.4 y Componentes opcionales reescritas; CodeGraph no ejecutado contra su instalador real | 2026-11-23 |

## Log
- L1: Que 'v-ia install --all' y un nuevo 'v-ia deps' instalen lo que falta (engram, codegraph, context7) en vez de fallar: Context7 por MCP remoto sin Node, Engram con go install/brew, CodeGraph con su instalador; nunca sudo; TUI muestra los pasos de instalación en la revisión.
- Risk: ítem 5 (instaladores y dependencias externas); anotado el 2026-11-25, a partir de las tareas de la ficha
