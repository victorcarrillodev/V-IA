---
name: odd-verify
description: Cerrar trabajo ODD solo con evidencia observada. Úsala antes de marcar [x] o declarar algo terminado.
license: "Apache-2.0"
metadata:
  author: "v-ia"
  version: "1.0"
---
# Cerrar con evidencia

1. Por cada tarea a marcar `[x]`, ejecuta la comprobación real y pega en `## Evidencia` una fila `| T-n | comando | resultado observado | fecha |`. El comando que cites debe ser uno que `v-ia review check` ejecute si quieres que `v-ia task verify` lo respalde.
2. **Pruebas**: un RED por cada regla pedida (míralo fallar por la razón correcta), GREEN, refactor; y por cada comando u opción existente que compartió código, una prueba de que su comportamiento previo se mantiene. **Diferencial** antes de terminar: ejecuta esos comandos contra una copia de la base en `mktemp -d` y contra el cambio, y compara stdout, stderr y código de salida. Que existan tests o un framework no basta para saber si aplican. Sin prueba útil, explica la excepción y haz una comprobación funcional.
3. Verifica la **petición**, no el trabajo del escritor: deriva sondas de la spec (`S#`), prueba los ejemplos literales contra un estado aislado y comprueba el invariante de que tras un comando rechazado los datos siguen intactos. Un fallo es bloqueante solo si se reproduce con entrada realista y no estaba ya en la base.
4. Ejecuta `v-ia task check` y `v-ia review check`; si hay revisión debida, `v-ia review status`. No cierres con alguno en rojo.
5. Cierra con `Risk: ítem N (motivo)` o `Risk: none`. Informa: verificado, fallido, omitido, pendiente. Un `[x]` no es aprobación ni recibo; una prueba ausente o rechazada prohíbe decir "terminado".
6. Commit/push/PR solo si el usuario lo pidió explícitamente en su mensaje.
