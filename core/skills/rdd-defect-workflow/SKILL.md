---
name: rdd-defect-workflow
description: Corregir un defecto con evidencia (reproducir, causa raíz, arreglo mínimo, comprobar). Úsala al arreglar un bug reportado o una regresión.
license: "Apache-2.0"
metadata:
  author: "v-ia"
  version: "1.0"
---
# Defectos con evidencia

1. **Reproduce primero** con un caso real (RED): un test o comando que falle por la razón correcta. Si no lo reproduces en la base limpia, dilo y no "arregles" a ciegas. El mecanismo que afirma el reporte es hipótesis; solo el síntoma es evidencia.
2. **Causa raíz** con `ruta:línea`. ¿Dónde más ocurre? Si es sistémico, ve a `systemic-issue-triage`. Un invariante causal por cambio, con rollback independiente.
3. **Pronóstico**: estima las líneas antes de editar; si superan ~400, planifica `chained-pr` o `work-unit-commits`.
4. **Arreglo mínimo que reduzca el sistema**: borrar o relajar suele ganar a añadir estado, flags o puertas. Si el arreglo es "nombrar la salida ejecutable", es un arreglo de mensaje: ejecuta el comando impreso para comprobarlo.
5. **Comprueba (GREEN)**: la prueba nueva, la de comportamiento previo del comando tocado y la suite. "Verde en tests pero roto en producción": sospecha que el test fue enseñado a coincidir; ejecuta el binario real.
6. Inventario de los flujos del usuario que toca el cambio, con al menos un control negativo cada uno. Un proxy sintético no prueba otro entorno.
7. Cierra con `Risk:` y un informe: defecto, causa, invariante, pruebas, rollback y decisiones que no eran tuyas. Commit/PR solo a petición.
