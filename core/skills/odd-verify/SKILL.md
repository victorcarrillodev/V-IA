---
name: odd-verify
description: Cerrar trabajo ODD solo con evidencia observada. Úsala antes de marcar [x] o declarar algo terminado.
---
# Cerrar con evidencia

1. Por cada tarea a marcar `[x]`, ejecuta la comprobación real y pega en `## Evidencia` una fila `| T-n | comando | resultado observado | fecha |`.
2. Si hay prueba ejecutable: mírala fallar antes de implementar y pasar después (test-first). Si no la hay, di por qué y haz una comprobación funcional.
3. Ejecuta `v-ia task check` y `v-ia review check`. No cierres con alguno en rojo.
4. Informa: verificado, fallido, pendiente. Un `[x]` no es aprobación ni recibo de revisión.
5. Commit/push/PR solo si el usuario lo pidió explícitamente en su mensaje.
