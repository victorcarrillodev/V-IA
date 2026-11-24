# Clasificación

| Señal | Ruta |
|---|---|
| "explica", "analiza", "propón", "sin modificar" | Lectura |
| Se entiende, riesgo contenido y se retomaría desde la petición + `git diff` | Acotado |
| No se retomaría desde la petición + diff: varias sesiones, esperas externas, entregables separados o requisitos que no sobreviven a una compactación | Seguimiento |
| Texto insuficiente | Pregunta el resultado esperado y si autoriza cambios |

**Acotado** exige las tres a la vez: *entendido* (lo demuestra un lote de lectura acotado: ~3 llamadas, ~10k tokens), *riesgo contenido* (no toca ningún ítem de riesgo alto) y *retomable desde el diff*. El número de archivos, comandos o pruebas, o una lista de pendientes, nunca decide el tamaño. Se hace en línea: sin ficha, sin subagentes y sin commits salvo que el usuario los pida.

**Riesgo alto** = toca alguno de estos seis ítems (lista cerrada): 1) datos o efectos irreversibles que no se pueden reconstruir; 2) seguridad; 3) contratos que otros consumen (API, flags de CLI, formatos de configuración, prompts espejados; añadir un flag no cuenta); 4) concurrencia; 5) entrega o entorno (instaladores, release, CI, dependencias); 6) ningún test detectaría una regresión. Antes de escribir `Risk: none`, comprueba si el diff cambia código que otro comportamiento no mencionado también usa: si lo cambia, es el ítem 3. Si juzgas que es alto, `v-ia review check --escalate-item N --escalate-reason "…"` lo eleva (nunca baja).

Un "sin modificar" explícito gana siempre sobre cualquier otra señal.
