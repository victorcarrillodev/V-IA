---
name: odd-resume
description: Retomar trabajo ODD interrumpido. Úsala al volver a una feature, tras compactación o cuando el usuario diga "retoma/continúa".
license: "Apache-2.0"
metadata:
  author: "v-ia"
  version: "1.0"
---
# Retomar una feature

1. Si hay memoria (Engram): recupera el contexto del proyecto, busca acotado por proyecto y feature (`odd/<feature>/tasks`) y lee la observación completa; luego el archivo `odd/tasks/<feature>.md` hasta `## Log`. No infieras el trabajo activo de la memoria global más reciente.
2. Contrasta con el código actual: `git log --oneline -10`, `git diff --stat` y los archivos citados. **Si el documento o la memoria contradicen al código, gana el código**: corrige el documento. Si dos versiones no se pueden reconciliar, conserva ambas y pregunta solo por el conflicto real.
3. Ejecuta `v-ia task status`, `v-ia task check` y, si hay recibo, `v-ia review status`.
4. Reabre solo las tareas invalidadas (con motivo en el Log); no rehagas lo demás. Un cambio de requisito añade su entrada literal al Log y reabre solo su tarea.
5. Continúa con la siguiente tarea sin pedir permiso si estaba autorizada; si no, di en una línea qué está hecho y verificado y qué sigue.
