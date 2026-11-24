---
name: chained-pr
description: Dividir un cambio grande en pull requests encadenados y revisables. Úsala cuando un cambio supere lo que se puede revisar de una vez.
---
# PRs encadenados

Solo con petición explícita del usuario de dividir o abrir PRs.

1. Decide el corte por **unidades de comportamiento** (código + pruebas + docs juntos), no por carpetas. Cada PR debe compilar y pasar sus pruebas por sí solo.
2. Ordénalos por dependencia: base primero. Cada PR apunta a la rama del anterior; el primero, a la rama por defecto.
3. Nómbralos con un prefijo común (`feat/x-1-base`, `feat/x-2-api`…) y en la descripción de cada uno enlaza el anterior y el siguiente, y di qué NO incluye.
4. Una unidad de riesgo alto va sola, para revisarla con `v-ia review check` y el revisor.
5. Tras fusionar uno, rebasa el siguiente sobre su nueva base. Nunca reescribas ramas ya publicadas sin que el usuario lo pida.
6. Anota en la ficha `odd/tasks/<feature>.md` qué PR contiene cada tarea.
