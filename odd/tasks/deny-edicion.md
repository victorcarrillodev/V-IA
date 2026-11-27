---
feature: deny-edicion
status: in-progress
created: 2026-11-25
updated: 2026-11-25
authorized-by: usuario / 2026-11-25 / punto 1; sin commit ni push
delivery: ask-on-risk
test-runner: go test -race ./...
---
## Specs
- S1: si, recuerda cada punto para hcerlos pero empieza por el punto 1

Puntos citados de la respuesta anterior (se hacen de uno en uno, cada uno cuando se pida o toque):
1. "**Reglas `deny` de edición.**"
2. "**Primera release.**"
3. "**Un subagente refutador** y uno por lente, sin escritura."
4. "**Identidad del revisor** que no dependa de un nombre declarado."
5. "**Modelo por subagente.**"

## Tareas
- [x] T-1 (S1) route=inline commit=- punto 1: reglas deny de edición

Pendientes (fuera de esta ficha, cada uno abre la suya): puntos 2, 3, 4 y 5.
<!-- route=inline|delegated(<motivo: exploración | escritores paralelos | respaldo de contexto | verificador independiente>); cada tarea de código cierra con "Risk: ítem N (motivo)" o "Risk: none" -->

## Evidencia
| Tarea | Comando / comprobación | Resultado observado | Fecha |
|---|---|---|---|
| T-1 | go test -run Deny ./internal/settings (antes); go test -race ./... (después) | RED: faltaban las 11 reglas Edit; GREEN: ok. Risk: ítem 2 (seguridad) | 2026-11-25 |

## Log
- L1: si, recuerda cada punto para hcerlos pero empieza por el punto 1
