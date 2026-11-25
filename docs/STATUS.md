# Estado de V-IA (2026-11-25)

Agentes principales: Claude Code, OpenCode, Codex y Antigravity. Referencia: Gentle AI v5.0.0 (comparación por documentación publicada y lectura parcial de código). Uso en `docs/manual.md`; evidencia en `odd/tasks/`.

## Por agente

| | Claude | OpenCode | Codex | Antigravity |
|---|---|---|---|---|
| Reglas ODD y delegación | sí | sí | sí | sí |
| Bloqueo por hooks | comprobado | comprobado (1.19.5) | instalable, sin verificar | no existe |
| Subagentes | sí | sí | no | no |
| Engram y CodeGraph configurados | sí | sí | sí (con `codex` en PATH) | sí |
| Uso de Engram/CodeGraph dentro de una sesión | sin probar | sin probar | sin probar | sin probar |

En total hay 17 agentes más `generic`; fuera de Claude Code y OpenCode no están verificados con el agente real.

## Corregido el 2026-11-25

- CI: pruebas y finales de línea para macOS y Windows (en verde en `4a1f20e`).
- `review check` no congelaba el candidato si `.v-ia/` o `.serena/` estaban ignoradas.
- Las lentes y el escalado se conservan mientras la revisión sigue abierta.
- Registros simultáneos de revisión: se perdían veredictos (3 de 4 en la reproducción). Ahora el recibo se bloquea.
- Revisión RDD completa con cuatro agentes independientes: `approved`. La orquestó una sesión de Claude Code a mano.
- Fichas antiguas con línea `Risk`.
- CI de Windows obligatoria; bloqueo del recibo y reemplazo del binario en ejecución probados en Windows (CI en verde en `9d962a1`), con las cuatro lentes aprobadas.

## Falta

1. Publicar la primera release y probar `install.sh` y `v-ia update` contra ella.
2. Instalar V-IA en el equipo y usarlo 1 o 2 semanas.
3. Comprobar que un agente sigue el protocolo de revisión por sí solo con `v-ia` instalado (la prueba con OpenCode y `big-pickle` no se completó).
4. Verificar hooks de Codex (requiere `codex login`) y su ejecutable en PATH.
5. Antigravity: bloqueo (capturar `PreToolUse` real; hooks en `~/.gemini/config/hooks.json`) y subagentes. El IDE está sin revisar.
6. Modelo por agente y subagentes en más clientes.
7. Medir el coste en tokens (solo hay tamaños en bytes).

## Riesgos conocidos

- Windows: si `update` no logra poner el binario nuevo ni restaurar el anterior, queda solo `v-ia.exe.old` y el error no lo dice.
- Windows: un `review status` o `verify` simultáneo podría hacer fallar el guardado del recibo (los lectores no toman el bloqueo; sin reproducir).
- Sin soporte de bloqueo en el sistema de archivos, cada operación de revisión espera 2 minutos y culpa a "otra operación en curso".
- Sin prueba: el bloqueo de `check` y el de `refute`, ni la vuelta atrás de `update` en Windows.
- El refutador se distingue por el nombre declarado, no por identidad.
- Si `.v-ia/`, `.atl/` o `.serena/` no están ignoradas, su contenido se guarda como objetos git sueltos al congelar el candidato.
- Un `review check` largo hace esperar a `ack`, `refute` y `reset`; pasados 2 minutos fallan y hay que reintentar.
- El respaldo no cubre plugins ni binarios que descargan Engram o CodeGraph; restaurar `~/.claude.json` tarde revierte estado de Claude Code.
- `install` ejecuta instaladores de terceros (omitible con `--codegraph=false`).
- La CI avisa de que las acciones fijadas usan Node 20, ya obsoleto.
