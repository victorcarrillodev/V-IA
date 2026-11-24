---
feature: paridad-gentle
status: done
created: 2026-11-23
updated: 2026-11-23
authorized-by: <usuario / fecha / alcance>
delivery: ask-on-risk
test-runner: <comando>
---
## Specs
- S1: Cerrar las brechas factibles frente a Gentle-AI: biblioteca de skills ampliada, v-ia update (auto-actualización verificada con sha256), CI en GitHub Actions. Fuera de alcance por no verificable aquí: agentes no instalados, Windows, benchmarks, telemetría, personas.

## Tareas
- [x] T-1 (S1) route=inline commit=- 7 skills nuevas (branch-pr, issue-creation, chained-pr, judgment-day, skill-creator, comment-writer, rdd-defect-workflow)
- [x] T-2 (S1) route=inline commit=- v-ia update con sha256, reemplazo atómico y pruebas con servidor simulado
- [x] T-3 (S1) route=inline commit=- CI en GitHub Actions (ubuntu y macos, fijado por SHA)
- [x] T-4 (S1) route=inline commit=- README y manual, con la brecha restante dicha con honestidad

## Evidencia
| Tarea | Comando / comprobación | Resultado observado | Fecha |
|---|---|---|---|
| T-1 | go test -count=1 ./... (incluye orphans_test) | ok: las 12 skills coinciden con skillNames y declaran su name | 2026-11-23 |
| T-2 | go test -count=1 ./internal/update; v-ia update --check | ok: hash manipulado rechazado, ruta peligrosa rechazada, reemplazo atómico; contra el repo real dice "no hay ninguna release publicada" | 2026-11-23 |
| T-3 | revisión del workflow | sin ejecutar en GitHub (aún no subido); gofmt/vet/test/sh -n se ejecutaron en local | 2026-11-23 |
| T-4 | revisión manual de README y manual | sección 4.4b, tabla de skills, límites frente a Gentle-AI | 2026-11-23 |

## Log
- L1: Cerrar las brechas factibles frente a Gentle-AI: biblioteca de skills ampliada, v-ia update (auto-actualización verificada con sha256), CI en GitHub Actions. Fuera de alcance por no verificable aquí: agentes no instalados, Windows, benchmarks, telemetría, personas.
