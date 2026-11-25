---
feature: riesgos-pendientes
status: done
created: 2026-11-25
updated: 2026-11-25
authorized-by: usuario / 2026-11-25 / corregir errores y riesgos; commit y push autorizados después ("si hazlo")
delivery: ask-on-risk
test-runner: go test -race ./...
---
## Specs
- S1: si, corrije primero el error y actualiza status, si hay mas errores o riesgos por correhir tambien hazlos

## Tareas
- [x] T-1 (S1) route=inline commit=- bloqueo del recibo entre check, ack, refute y reset
- [x] T-2 (S1) route=inline commit=- STATUS, README, manual y líneas Risk de fichas antiguas
- [x] T-3 (S1) route=inline commit=- reemplazo del binario en Windows y CI de Windows obligatoria 
<!-- route=inline|delegated(<motivo: exploración | escritores paralelos | respaldo de contexto | verificador independiente>); cada tarea de código cierra con "Risk: ítem N (motivo)" o "Risk: none" -->

## Evidencia
| Tarea | Comando / comprobación | Resultado observado | Fecha |
|---|---|---|---|
| T-1 | go test -count=3 -run Concurrent ./internal/review (antes); go test -race -count=20 -run 'Concurrent|Busy' (después) | RED: 1 veredicto de 4; GREEN: ok. Risk: ítem 4 (concurrencia) | 2026-11-25 |
| T-2 | go run ./cmd/v-ia task check; go test ./... | fichas con línea Risk; pruebas en verde. Risk: none | 2026-11-25 |
| T-3 | v-ia review ack (4 lentes) y verify; git push; gh run view 45710702861 | approved, 0 bloqueantes; ubuntu, macos y windows en success. Risk: ítem 5 (CI y update) | 2026-11-25 |

## Log
- L2: "si hazlo" (a: lanzar la revisión por lentes y, si aprueba, commit y push)
- L1: si, corrije primero el error y actualiza status, si hay mas errores o riesgos por correhir tambien hazlos
