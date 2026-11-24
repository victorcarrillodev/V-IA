---
name: judgment-day
description: Revisión adversarial ciega con dos jueces y como máximo dos rondas de corrección. Úsala solo si el usuario pide una revisión a fondo o una segunda opinión de un objetivo concreto.
license: "Apache-2.0"
metadata:
  author: "v-ia"
  version: "1.0"
---
# Judgment Day

Sustituye a la revisión 4R ordinaria para ese objetivo (no corras ambas). No emite recibo ni autoriza commit, push, PR o release. Cuesta dos subagentes: solo a petición del usuario o con riesgo alto.

1. Congela UN objetivo inmutable (diff o lista de rutas) y resuelve las skills del proyecto; pasa las mismas rutas a todos.
2. Lanza `v-ia-judge-a` y `v-ia-judge-b` EN PARALELO con el mismo alcance y criterios y nada más (sin tu opinión ni conclusiones). Espera a los dos: nunca aceptes un juicio parcial. No lances refutador: la corroboración es el acuerdo entre los dos jueces.
3. Fusiona tú en un ledger congelado: id, ubicación, severidad (BLOCKER | CRITICAL | WARNING | SUGGESTION), estado (`open | fixed | verified | refuted | wont-fix | info`) y evidencia. WARNING y SUGGESTION nunca quedan `open`: se informan una vez como `info`.
4. Tabla de decisión:
   - Severo confirmado por **ambos** → pregunta al usuario antes de la ronda 1 y corrige.
   - Lo ve **uno solo** → `suspect`: no se corrige automáticamente; verifícalo tú leyendo el código.
   - Los jueces **se contradicen** → decisión humana explícita.
   - Un hallazgo sin caso concreto que falle se descarta.
5. Corrección: `v-ia-fixer` solo con los IDs severos confirmados, con unidades atómicas (test enfocado, evidencia y límite de rollback) y `## Allowed edit surfaces`. No añade hallazgos ni refactoriza.
6. Re-juicio **acotado**: ambos jueces ven solo el ledger congelado más el delta de la corrección (verifican cada resolución y las regresiones que causó). Como máximo **dos rondas** de corrección y dos re-juicios; nunca una tercera ni se reinicia el presupuesto.
7. Veredicto terminal: `APPROVED` o `ESCALATED` (queda algún severo tras la ronda 2). Informa: identidad del objetivo, ronda, confirmados / sospechosos / contradicciones / info, qué se corrigió y qué quedó sin verificar. Termina con una línea: `JUDGMENT: APPROVED` o `JUDGMENT: ESCALATED`.
