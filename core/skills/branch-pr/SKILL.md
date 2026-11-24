---
name: branch-pr
description: Preparar rama y pull request. Úsala solo cuando el usuario pida abrir un PR o preparar la entrega a una rama.
---
# Rama y PR

Solo con petición explícita del usuario de crear la rama o el PR. Un recibo RDD `pass` no lo autoriza.

1. Nunca trabajes sobre la rama por defecto: crea `tipo/descripcion-corta` (`feat/`, `fix/`, `docs/`…).
2. Lee la política del repositorio destino antes de escribir: `CONTRIBUTING.md`, plantilla de PR (`.github/pull_request_template.md`), etiquetas y convención de títulos. Si existen, mandan sobre esta skill.
3. Antes de abrir: `v-ia review check` en verde y árbol limpio. Si el riesgo es alto, que el revisor haya registrado su veredicto.
4. Título en Conventional Commits. Descripción con: **qué** cambia, **por qué**, **cómo se probó** (comandos y resultado observado, no "probado") y **riesgos / qué no cubre**.
5. Un PR = una intención revisable. Si crece demasiado, usa la skill `chained-pr`.
6. Enlaza el issue o la ficha `odd/tasks/<feature>.md`. No fuerces el push ni toques ramas ajenas.
