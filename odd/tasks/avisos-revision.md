---
feature: avisos-revision
status: done
created: 2026-11-25
updated: 2026-11-25
authorized-by: usuario / 2026-11-25 / corregir los avisos listados; sin commit ni push
delivery: ask-on-risk
test-runner: go test -race ./...
---
## Specs
- S1: corrige los errores existentes

## Tareas
- [x] T-1 (S1) route=inline commit=- avisos de update, recibo y bloqueo, y pruebas que faltaban
<!-- route=inline|delegated(<motivo: exploración | escritores paralelos | respaldo de contexto | verificador independiente>); cada tarea de código cierra con "Risk: ítem N (motivo)" o "Risk: none" -->

## Evidencia
| Tarea | Comando / comprobación | Resultado observado | Fecha |
|---|---|---|---|
| T-1 | go test -run 'Swap|CheckAndRefute|Unsupported|Transient' (antes); go test -race ./... y go vet cruzado (después) | RED: 3 fallos esperados; GREEN en Linux; Windows y macOS sin ejecutar. Risk: ítem 4 (bloqueo y guardado) e ítem 5 (update) | 2026-11-25 |

## Log
- L1: corrige los errores existentes
