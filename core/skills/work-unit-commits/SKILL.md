---
name: work-unit-commits
description: Commits por unidad de trabajo revisable. Úsala cuando el usuario pida commit o al entregar trabajo por tareas.
---
# Commits por unidad de trabajo

Solo con petición explícita del usuario de hacer commit.

- Un commit = una unidad coherente (~400 líneas editadas como guía, no como tope) con sus tests y docs. Sin dividir artificialmente.
- Si estás en la rama por defecto, crea antes una rama de feature.
- Mensaje Conventional Commits (`feat:`, `fix:`, `docs:`…).
- Una unidad cuyo riesgo sea alto va en su propio commit, para poder revisarla sola.
- Antes: `v-ia review check` en verde. Anota en la ficha el commit que contiene cada tarea.
- Push o PR solo si el usuario los pide además del commit.
