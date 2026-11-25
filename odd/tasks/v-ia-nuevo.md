---
feature: v-ia-nuevo
status: in-progress
created: 2026-11-24
updated: 2026-11-24
authorized-by: usuario / 2026-11-24 / "ok me parece bien entonces, comienza" (ficha y diseño de doctor; sin commit ni push)
delivery: ask-on-risk
test-runner: go test ./...
---
## Specs
- S1: "quiero poder economizar en tokens, poder hacer que funcionen todos con ODD y incrementar el uso profesional de la ia con reglas de conductas adecuadas para el uso de estas en mi trabajo"
- S2: "mi trabajo es programar y otras cosas diferentes, y actualmente uso opencode, agy cli, claude y codex"
- S3: "aun asi quiero mi propia herramienta [...] revisar mi github y traerlo, ademas quiero mi propia herramienta para aprender como esta hecho, que se hace y como poder mejorarla"
- S5: "quiero que puedas igualar el desempeño en cada una y has revisiones siempre empezando por los test donde los necesite"
- S6: "quiero que hagas una comparacion exahustiva en en el recibo RDD y fichas odd con gentle ai y mejores cada ambito que sea necesario y tambien el punto de memoria y mcp, ademas quiero que incluyas codegraph en el stack"
- S7: "quiero que te enfoques en la orquestacion y delegacion para poder refinarlo, adeams emgram y code graph no puede usarse en todos?"
- S8: "No tiene un refutador que descarte falsos positivos [...] El recibo no guarda la petición original [...] V-IA omita el suyo cuando Engram está instalado [...] haz las pruebas y revisa que funcionen, hazlo."
- S9: "haz las pruebas y mitiga los riesgos" / "ahora haz un archvo de status de como queda hasta ahora v-ia y que falta, haz  commit y push"
- S4: "tener una herramienta al nivel de gentle ai para poder hacer mas profesional mi herramienta"

## Criterios (propuestos y aceptados)
- Partir del código anterior (commit 3a8d6e9) y pulirlo comparando con Gentle AI (sustituye a "empezar de cero", ver L4).
- Respaldo antes de escribir, escritura atómica, `--dry-run`, `restore`; `sync` repetido no cambia nada.
- Cada regla con prueba, sin tocar la configuración real; CI en cada push; instalación en un comando con checksum.
- Errores que dicen qué hacer; README, ayuda y decisiones documentadas.
- Fuera al principio: TUI, telemetría, autoactualización, otros agentes.
- No se pasa de etapa sin pruebas en verde, resultado comprobado y que el usuario entienda la parte.

## Tareas
- [x] T-1 (S3) retirar la instalación de V-IA del equipo, con respaldo
- [x] T-2 (S3) comprobar que el código anterior compila y pasa sus pruebas
- [x] T-3 (S4) comparar con Gentle AI en los cuatro agentes y priorizar mejoras
- [x] T-4 (S2) corregir detección de Antigravity CLI (binario `agy`, `~/.gemini/antigravity-cli`)
- [x] T-5 (S2) los 4 agentes por defecto; sync/uninstall respetan lo instalado
- [x] T-6 (S5) doctor informa los 4 y no promete bloqueo sin verificar
- [x] T-7 (S5) verificar el bloqueo con OpenCode real
- [ ] T-10 (S5) verificar hooks con Codex real (falta `codex login`)
- [x] T-11 (S1) el estado local de `.v-ia/` no entra en commits
- [ ] T-8 (S5) bloqueo y subagentes en Antigravity
- [ ] T-9 (S1) Engram/MCP en los 4
- [x] T-12 (S6) comparar y mejorar fichas ODD
- [x] T-13 (S6) comparar y mejorar recibo RDD
- [x] T-14 (S6, absorbe T-9) memoria y MCP en los 4
- [x] T-15 (S6) CodeGraph en el stack
- [x] T-16 (S7) reglas de delegación siempre cargadas, según capacidad de cada agente
- [ ] T-17 (S7) subagentes en Codex y Antigravity
- [x] T-18 (S8) refutador de falsos positivos
- [x] T-19 (S8) petición congelada en el recibo
- [x] T-20 (S8) no duplicar el protocolo de memoria con Engram
- [x] T-21 (S9) respaldo y re-planificación con instaladores externos; código bloqueado no reutilizable
- [ ] T-22 (S9) revisión completa con un agente de IA real
- [x] T-23 (S9) docs/STATUS.md

## Evidencia
| Tarea | Comando / comprobación | Resultado observado | Fecha |
|---|---|---|---|
| T-1 | `command -v v-ia`; `grep -il v-ia` en las 4 configs | no está; sin menciones; respaldo ~/respaldo-via-2026-11-24.tar.gz (48 entradas) | 2026-11-24 |
| T-2 | `go build ./...`, `go vet ./...`, `gofmt -l .`, `go test -race ./...`, GOOS=windows/darwin | todo sin errores; race exit=0; `doctor` ejecutado en HOME temporal | 2026-11-24 |
| T-3 | lectura de adapters/lifecycle/hooks de V-IA y adapters/assets de Gentle AI; `wc -c` (comparación estática, sin ejecutar los agentes) | hooks de codex/gemini marcados "SIN VERIFICAR"; antigravity sin hooks ni detección de `agy`; reglas ~9 KB vs orquestador Gentle 12-19 KB | 2026-11-24 |
| T-4 | `go test ./internal/adapters` | RED: 2 fallos (agy, antigravity-cli); GREEN | 2026-11-24 |
| T-5 | `go test ./cmd/v-ia` + `go test -race ./...` | RED observado en instalación por defecto y sync/uninstall; GREEN, race exit=0 | 2026-11-24 |
| T-6 | `go test ./internal/doctor` | RED y luego GREEN | 2026-11-24 |
| T-7 | `opencode run -m opencode/big-pickle` 1.19.5, config y repo temporales | write bloqueado en lectura; edición permitida; commit pedido en pasos.txt bloqueado | 2026-11-24 |
| T-11 | `go test ./internal/hook` + `go test -race ./...` | RED y luego GREEN, race exit=0 | 2026-11-24 |
| T-12 | `go test ./internal/task` + `go test -race ./...` | RED (ID repetido, done, cita literal) y GREEN | 2026-11-24 |
| T-13 | `go test ./internal/review` + `go test -race ./...` | RED (--base, veredicto sobrescrito, presupuesto) y GREEN | 2026-11-24 |
| T-15 | `go test ./cmd/v-ia ./internal/extras`; `v-ia install --dry-run --engram` en HOME temporal | RED y GREEN; plan para los 4; sin ejecutar instaladores reales | 2026-11-24 |
| T-16 | `go test ./internal/adapters` y `go test ./...` | RED y GREEN | 2026-11-24 |
| T-14 | `engram setup` (4) y `codegraph install --target=claude,opencode,codex,antigravity` en HOME temporal | exit 0; ambos presentes en las 4 configuraciones; uso en sesión sin probar | 2026-11-24 |
| T-18 | `go test -race ./...`; `v-ia review ack/refute/verify` en repo temporal | mismo revisor rechazado; otro refutador -> verify=0 | 2026-11-24 |
| T-19 | `go test -race ./...`; `v-ia review check --request` | petición visible en `review status --json` | 2026-11-24 |
| T-20 | `v-ia install --engram` real en HOME temporal, 4 agentes; `sync --dry-run`; `doctor` | protocolo completo ausente en los 4; 0 cambios; bloques al día | 2026-11-24 |
| T-21 | `go test ./cmd/v-ia ./internal/review` y `go test -race ./...` | RED (3 fallos) y GREEN | 2026-11-24 |
| T-23 | revisión del archivo contra la evidencia | creado | 2026-11-24 |

## Siguiente paso
Ver docs/STATUS.md, sección "Falta".

## Log
- L1: "ok me parece bien entonces, comienza"
- L2: Entorno 2026-11-24: go 1.27.4; opencode 1.19.5, agy 1.5.16, claude 2.1.332 en PATH; `codex` no está en PATH (existe `~/.codex`).
- L3: Hay un `~/.local/bin/v-ia` anterior instalado, con hooks activos en la configuración global.
- L4: "quiero que borres todo lo de v-ia de mi equipo, ademas revises si el codigo anterior funciona y vayas puliendo a la par con gentle ai para poder irlo mejorando"
- Risk: none (no hubo cambios de código)
- Risk: ítem 3 (cambia el valor por defecto de --agent y la salida de doctor)
- L5: "tinenes todos los permisos que necesites y codex lo uso ocacionalmente en cli [...] antigravity uso el cli y su ide"
- L6: "quiero que hagas una comparacion exahustiva en en el recibo RDD y fichas odd [...] ademas quiero que incluyas codegraph en el stack"
- Risk: ítem 3 (formato de fichas y recibo, preset standard) e ítem 5 (instala CodeGraph)
- L7: "quiero que te enfoques en la orquestacion y delegacion [...] emgram y code graph no puede usarse en todos?"
- Risk: ítem 3 (cambian las reglas que se instalan en cada agente)
- L8: "no se a que te refieres, explicame mejor y haz las pruebas y revisa que funcionen, hazlo."
- Risk: ítem 3 (formato del recibo, flags nuevos de CLI, reglas instaladas)
- L9: "ahora haz un archvo de status [...] haz  commit y push"
- Risk: ítem 3 e ítem 5
