---
name: branch-pr
description: Preparar rama y pull request según la política del repositorio destino. Úsala solo cuando el usuario pida abrir un PR o preparar la entrega a una rama.
license: "Apache-2.0"
metadata:
  author: "v-ia"
  version: "1.0"
---
# Rama y PR

Solo con petición explícita. No hagas commit, push, PR ni merge por tu cuenta.

1. **Destino**: confirma con el usuario el repositorio base exacto (no el `origin` de un fork), la operación y la credencial antes de leer contra el host; no sondees credenciales ambientales.
2. **Política del repo manda**: plantilla de PR, etiquetas y reglas de CI las dicta el repositorio destino. No inventes etiquetas ni impongas `status:approved`, `type:*`, un límite de líneas o una regex solo porque esta skill esté instalada. Las etiquetas protegidas requieren instrucción humana directa y permiso de mantenimiento.
3. **Rama** desde la base actualizada; título Conventional Commits con un solo ámbito; descripción con el porqué, evidencia observada y qué NO incluye.
4. **Issue**: pregunta si el PR cierra el issue (`Closes #N`) o solo lo referencia (`Refs #N`); no lo infieras.
5. **Checkboxes**: marca `[x]` solo con evidencia observada; lo no soportado queda sin marcar y descrito como pendiente.
6. **Fallos preexistentes**: solo se afirman comparando el mismo comando y entorno en una base limpia aislada (nunca con `stash/pop`); si no, "baseline no verificado".
7. **CI requerido**: se establece desde rulesets o branch protection; si no sabes si es obligatorio, no está listo para merge.
8. Si supera ~400 líneas, aplica `chained-pr`. Informa: destino y base exactos, requisitos de la política, borrador, checks observados y evidencia que falta. Un recibo RDD `pass` no autoriza abrir el PR.
