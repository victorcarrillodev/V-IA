---
feature: standards
status: done
created: 2026-11-23
updated: 2026-11-23
authorized-by: <usuario / fecha / alcance>
delivery: ask-on-risk
test-runner: <comando>
---
## Specs
- S1: Implementar v-ia standards (init|check|install|off|status): reglas path/deny locales sin tokens y reglas ai con una llamada por commit (claude -p, modelo barato, JSON), caché por hash de fragmento+regla, fail-open, pre-commit de Git con marcadores, aviso por defecto y --strict. Parte determinista también en review check.

## Tareas
- [x] T-1 (S1) route=inline commit=- Reglas path/deny/ai: parser, globs, diff de staging, evaluación local
- [x] T-2 (S1) route=inline commit=- Revisión ai: una llamada, caché, presupuesto, fail-open
- [x] T-3 (S1) route=inline commit=- Hook pre-commit con marcadores y comandos init|check|install|off|status
- [x] T-4 (S1) route=inline commit=- Aviso local en review check, README y manual

## Evidencia
| Tarea | Comando / comprobación | Resultado observado | Fecha |
|---|---|---|---|
| T-1 | go test -count=1 ./internal/standards | ok: plantilla válida, 10 casos de error, globs, diff, deterministas | 2026-11-23 |
| T-2 | go test + v-ia standards check con Claude real en repo temporal | ok: detectó error sin envolver (a.go:9) en regla ai; segunda pasada 4 ms desde caché; fallos no cachean | 2026-11-23 |
| T-3 | hook en repo temporal: install, install --strict, SKIP, off | aviso rc=0; strict rc=1; SKIP rc=0; off conserva lo tuyo; sin claude: avisa y sigue | 2026-11-23 |
| T-4 | go test -count=1 ./... | 12 paquetes ok; docs añadidas | 2026-11-23 |

## Log
- L1: Implementar v-ia standards (init|check|install|off|status): reglas path/deny locales sin tokens y reglas ai con una llamada por commit (claude -p, modelo barato, JSON), caché por hash de fragmento+regla, fail-open, pre-commit de Git con marcadores, aviso por defecto y --strict. Parte determinista también en review check.
