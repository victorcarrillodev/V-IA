---
feature: revision-y-ci
status: in-progress
created: 2026-11-25
updated: 2026-11-25
authorized-by: usuario / 2026-11-25 / puntos 1-3, incluye commit y push
delivery: ask-on-risk
test-runner: go test -race ./...
---
## Specs
Puntos citados de la propuesta inmediatamente anterior:
- S1: "Commit, push y CI en verde."
- S2: "Una revisión RDD completa con un modelo capaz."
- S3: "Conservar las lentes elegidas en `review check`. Al repetirlo cambió de 1 a 4, y un agente también podría reducirlas."

## Tareas
- [x] T-1 (S3) route=inline commit=- conservar lentes (y el escalado) dentro de una revisión abierta
- [x] T-2 (S2) route=delegated(verificador independiente) commit=- revisión por lente del cambio completo
- [ ] T-3 (S1) route=inline commit=- commit, push y CI en verde
<!-- route=inline|delegated(<motivo: exploración | escritores paralelos | respaldo de contexto | verificador independiente>); cada tarea de código cierra con "Risk: ítem N (motivo)" o "Risk: none" -->

## Evidencia
| Tarea | Comando / comprobación | Resultado observado | Fecha |
|---|---|---|---|
| T-1 | go test -count=1 -run 'Lenses|Escalation|Fresh|Repeat' ./internal/review (antes); go test -race ./... (después) | RED: 3 fallos esperados (lentes 1→4, reducción aceptada, escalado perdido); GREEN: todo ok. Risk: ítem 3 (formato/comportamiento del recibo) | 2026-11-25 |
| T-2 | v-ia review ack (4 lentes, agentes independientes Opus); segunda pasada reliability+readability; v-ia review verify | approved en ambas pasadas, 0 bloqueantes; recibo válido sobre el candidato 7596259e7148 (anterior a esta anotación). Risk: ítem 3 | 2026-11-25 |

## Log
- L1: entonces has los puntos 1 2 y 3
