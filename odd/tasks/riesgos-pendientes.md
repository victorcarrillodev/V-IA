---
feature: riesgos-pendientes
status: in-progress
created: 2026-11-25
updated: 2026-11-25
authorized-by: usuario / 2026-11-25 / corregir errores y riesgos; sin commit ni push
delivery: ask-on-risk
test-runner: go test -race ./...
---
## Specs
- S1: si, corrije primero el error y actualiza status, si hay mas errores o riesgos por correhir tambien hazlos

## Tareas
- [x] T-1 (S1) route=inline commit=- bloqueo del recibo entre check, ack, refute y reset
- [x] T-2 (S1) route=inline commit=- STATUS, README, manual y líneas Risk de fichas antiguas
- [ ] T-3 (S1) route=inline commit=- reemplazo del binario en Windows y CI de Windows obligatoria (falta verlo en CI)
<!-- route=inline|delegated(<motivo: exploración | escritores paralelos | respaldo de contexto | verificador independiente>); cada tarea de código cierra con "Risk: ítem N (motivo)" o "Risk: none" -->

## Evidencia
| Tarea | Comando / comprobación | Resultado observado | Fecha |
|---|---|---|---|
| T-1 | go test -count=3 -run Concurrent ./internal/review (antes); go test -race -count=20 -run 'Concurrent|Busy' (después) | RED: 1 veredicto de 4; GREEN: ok. Risk: ítem 4 (concurrencia) | 2026-11-25 |
| T-2 | go run ./cmd/v-ia task check; go test ./... | fichas con línea Risk; pruebas en verde. Risk: none | 2026-11-25 |

## Log
- L1: si, corrije primero el error y actualiza status, si hay mas errores o riesgos por correhir tambien hazlos
