# Estado de V-IA (2026-11-24)

Agentes principales: Claude Code, OpenCode, Codex y Antigravity. Referencia: Gentle AI (comparación parcial, no función por función de todo su código). Detalle y evidencia en `odd/tasks/v-ia-nuevo.md`; uso en `docs/manual.md`.

## Por agente

| | Claude | OpenCode | Codex | Antigravity |
|---|---|---|---|---|
| Reglas ODD y delegación | sí | sí | sí | sí |
| Bloqueo por hooks | comprobado | comprobado (1.19.5) | instalable, sin verificar | no existe |
| Subagentes | sí | sí | no | no |
| Engram y CodeGraph configurados | sí | sí | sí (con `codex` en PATH) | sí |
| Uso de Engram/CodeGraph dentro de una sesión | sin probar | sin probar | sin probar | sin probar |

## Hecho en esta ronda

- Los 4 agentes por defecto; `sync` y `uninstall` respetan lo instalado.
- Detección de Antigravity CLI (`agy`).
- `doctor` y las reglas solo prometen bloqueo donde se comprobó.
- `.v-ia/` ya no mete su estado local en commits.
- Fichas ODD: IDs repetidos, cierre incompleto y cita literal.
- Recibo RDD: `--base`, veredictos no sobrescribibles, presupuesto de corrección, código bloqueado no reutilizable, petición congelada y refutador.
- CodeGraph en el preset `standard`; Engram para los 4.
- Respaldo antes de instaladores externos; corregido que V-IA pisaba lo que ellos escribían.
- Sin protocolo de memoria duplicado cuando hay Engram.

Verificado con `go vet`, `go test -race ./...` y compilación para Windows y macOS.

## Falta

1. **Revisión con IA real: probada una vez y no completada.** Con OpenCode y el modelo gratuito `big-pickle`, el revisor no registró ningún veredicto y el refutador no refutó nada. El recibo siguió bloqueando la entrega (`checking`).
2. **La selección de lentes no se conserva.** En esa prueba un `review check` repetido sin `--lenses` cambió de 1 a 4 lentes. En sentido contrario, un agente podría reducirlas.
3. Verificar hooks de Codex (requiere `codex login`) y su ejecutable en PATH.
4. Antigravity: bloqueo (capturar `PreToolUse` real; hooks en `~/.gemini/config/hooks.json`) y subagentes. La cuota de `agy` se restablece hacia el 2026-12-01. El IDE está sin revisar.
5. Instalar V-IA en el equipo y usarlo 1 o 2 semanas.
6. Medir el coste en tokens (solo hay tamaños en bytes).
7. Cinco fichas antiguas de `odd/tasks/` fallan `task check` por no tener línea `Risk`.
8. README sin revisar tras estos cambios.

## Riesgos conocidos

- El refutador se distingue por el nombre declarado, no por identidad.
- El respaldo no cubre plugins ni binarios que descargan Engram o CodeGraph; restaurar `~/.claude.json` tarde revierte estado de Claude Code.
- `install` ejecuta instaladores de terceros (omitible con `--codegraph=false`).
