---
name: rdd-review
description: Revisión RDD de un cambio terminado. Úsala antes de reportar completo un trabajo, cuando RDD esté activo.
---
# RDD: revisar según el riesgo

1. `v-ia review mode status`. Si está OFF, solo comprobaciones ordinarias.
2. `v-ia review check`: ejecuta build/tests, evalúa el riesgo (`passive` | `medium` | `high`) y emite un recibo ligado al árbol exacto.
3. **passive**: nada más. **medium**: ofrece una revisión al usuario; solo si acepta, sigue el paso 4. **high** (`review_due`): paso 4 obligatorio para poder entregar.
4. Lanza el subagente `v-ia-reviewer` (independiente: no el que escribió el cambio). Él inspecciona el diff con las lentes 4R y registra su veredicto con `v-ia review ack`.
5. Si pide cambios: una sola corrección acotada y vuelve a `review check` (el árbol cambió, el recibo anterior ya no vale).
6. El recibo `pass` **no autoriza** commit, push ni PR: eso lo decide el usuario.
