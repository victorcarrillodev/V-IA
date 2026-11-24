---
name: rdd-defect-workflow
description: Corregir un defecto con evidencia (reproducir, arreglar, comprobar). Úsala cuando el usuario reporte un bug o un fallo a corregir.
---
# Defecto con evidencia

1. **Reproduce primero.** Escribe o ejecuta una comprobación que falle por la causa descrita y guarda su salida literal (RED). Si no puedes reproducirlo, dilo; no arregles a ciegas.
2. **Causa raíz.** Busca por qué ocurre, no solo dónde se ve. Anota la hipótesis y cómo la comprobaste; descarta las que no sobrevivieron.
3. **Arreglo mínimo** que cubra la causa, sin refactors ajenos. Si el mismo error puede estar en otros sitios, búscalos y di cuáles tocaste.
4. **Comprueba (GREEN):** la reproducción pasa y el resto de pruebas sigue en verde. Registra comando y resultado en `## Evidencia` de la ficha (en trabajos pequeños, en tu informe).
5. `v-ia review check` antes de reportar. Un defecto en seguridad, datos o contratos compartidos es riesgo alto: pide el revisor.
6. Informa con honestidad: qué quedó verificado, qué no y qué podría reaparecer. Commit o push solo si el usuario lo pide.
