---
name: odd-resume
description: Retomar trabajo ODD interrumpido. Úsala al volver a una feature, tras compactación o cuando el usuario diga "retoma/continúa".
---
# Retomar una feature

1. Lee `odd/tasks/<feature>.md` completo (Specs, Tareas, Evidencia, Log). Si hay memoria (Engram), busca también `odd/<feature>/tasks`.
2. Contrasta con el código actual: `git log --oneline -10`, `git diff --stat` y los archivos citados. **Si el documento o la memoria contradicen al código, gana el código**: corrige el documento.
3. Ejecuta `v-ia task status` y `v-ia task check`.
4. Reabre solo las tareas invalidadas (con motivo en el Log); no rehagas lo demás.
5. Di en una línea: qué está hecho y verificado, qué sigue.
