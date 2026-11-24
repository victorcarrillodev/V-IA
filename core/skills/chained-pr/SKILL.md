---
name: chained-pr
description: Dividir un cambio de más de ~400 líneas en PRs encadenados revisables. Úsala cuando el cambio previsto no cabe en un PR o el usuario pida PRs encadenados.
license: "Apache-2.0"
metadata:
  author: "v-ia"
  version: "1.0"
---
# PRs encadenados

- **Umbral**: ~400 líneas (`additions+deletions`) y revisable en ≤60 minutos. El presupuesto limita el corte, nunca el código: no borres comentarios, docs ni tests ni comprimas código para encajar.
- **Un solo intento honesto de corte** por límites de comportamiento. Si no hay un corte cohesivo dentro del presupuesto, entrega el mejor y reporta el exceso (y que necesita una excepción de tamaño del repo).
- **Elige una estrategia y no las mezcles**: *Stacked a main* (cada PR sobre el anterior, mergeables en orden) o *Feature Branch Chain* con un PR "tracker" en draft/no-merge (el hijo 1 apunta al tracker; los siguientes, a su padre inmediato).
- **Cada PR hijo** compila y pasa sus pruebas por sí solo, y lleva "Contexto de la cadena": cadena, tracker, posición, base, depende de, siguiente, qué incluye y qué NO incluye, con un diagrama que marca el PR actual.
- Un diff contaminado es un bug de base: retargetea o rebasa; no lo tapes en la descripción.
- Riesgo alto en su propio PR. Push, PR y merge los decide el usuario (`branch-pr`).
