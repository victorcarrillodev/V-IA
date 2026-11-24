---
name: judgment-day
description: Revisión adversarial con dos jueces independientes en paralelo. Úsala cuando el usuario pida una revisión a fondo o una segunda opinión de un cambio crítico.
---
# Juicio: dos revisores independientes

Cuesta dos subagentes; úsala solo si el usuario la pide o el riesgo es alto (`v-ia review check`).

1. Fija el objetivo exacto: `git diff` o la lista de archivos. Los dos jueces reciben **el mismo** objetivo y nada más (sin tu opinión ni conclusiones previas).
2. Lanza dos subagentes `v-ia-reviewer` en paralelo, cada uno por separado. Pídeles que busquen fallos reales (con caso concreto y `ruta:línea`), no estilo.
3. Compara en tabla: hallazgos que coinciden (alta confianza), los que solo ve uno (verifícalos tú leyendo el código) y los descartados.
4. Un hallazgo sin caso concreto que falle se descarta. No promedies opiniones: comprueba.
5. Corrige solo lo confirmado, en una pasada acotada, y vuelve a `v-ia review check` (el árbol cambió).
6. Informa: confirmados, descartados con motivo, y qué quedó sin verificar. El juicio no autoriza commit ni push.
