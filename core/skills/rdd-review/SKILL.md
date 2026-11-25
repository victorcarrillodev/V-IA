---
name: rdd-review
description: Revisión RDD de un cambio terminado. Úsala antes de reportar completo un trabajo, cuando RDD esté activo.
license: "Apache-2.0"
metadata:
  author: "v-ia"
  version: "1.0"
---
# RDD: revisar según el riesgo

La revisión independiente es un par de ojos extra para trabajo de riesgo alto, nunca un ritual por candidato.

1. `v-ia review mode status`. Si está OFF, solo comprobaciones ordinarias.
2. `v-ia review check --request "<petición literal>"` (o `--request-file`): congela la petición, corre build/tests, riesgo (`passive` | `medium` | `high`) y recibo ligado al cambio exacto (dos árboles git congelados). Si tu juicio es que es alto por un ítem de `classify.md`: `--escalate-item N --escalate-reason "…"` (solo sube). En medium/high puedes elegir lentes con `--lenses a,b --lenses-reason "…"`; por defecto high = 4 lentes (`risk`, `resilience`, `readability`, `reliability`), medium = `reliability`, passive = ninguna.
3. `v-ia review status --json` dice qué falta (`working | checking | ready | needs_decision`) y la `next_transition`. Solo `high` (o evaluación fallida) exige revisión; en medium ofrécela al usuario.
4. Por CADA lente pendiente, lanza `v-ia-reviewer` (independiente, no el que escribió el cambio) con la lente. Revisa el candidato congelado (`v-ia review inspect`), cubre todo el manifiesto de rutas y registra su resultado con `v-ia review ack --input -` (JSON estricto). **El veredicto lo calcula V-IA**: solo bloquean BLOCKER/CRITICAL causados por el cambio y con prueba; lo preexistente o no probado no bloquea.
5. Si queda un bloqueante: **una sola corrección acotada** (`v-ia-fixer` o tú, dentro de las rutas del manifiesto) y de nuevo `v-ia review check`. Si el segundo intento vuelve a fallar, el estado es `escalated` = *necesita tu decisión*: preséntale al usuario el bloqueo exacto y para; `v-ia review reset` solo lo ejecuta él.
   - Si el bloqueante parece un **falso positivo**, antes de corregir lanza un refutador distinto del revisor: intenta reproducirlo y registra `v-ia review refute --input -` con `{"candidate_id":"…","refuter":"…","results":[{"id":"risk#1","verdict":"refuted|upheld","evidence":["…"]}]}`. Una sola vez por candidato, respondiendo a todos los bloqueantes; ante la duda, `upheld`. Los IDs salen de `v-ia review status --json`.
6. El recibo `pass` y la aprobación **no autorizan** commit, push ni PR: eso lo decide el usuario.
