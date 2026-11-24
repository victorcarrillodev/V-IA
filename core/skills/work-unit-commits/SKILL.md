---
name: work-unit-commits
description: Dividir el trabajo en commits por unidad de trabajo revisable. Úsala cuando el usuario pida commits o PR y el cambio previsto supere ~400 líneas.
license: "Apache-2.0"
metadata:
  author: "v-ia"
  version: "1.0"
---
# Commits por unidad de trabajo

Solo con petición explícita de commit. Una unidad = un comportamiento con su código, pruebas y documentación juntos; no commitees por tipo de archivo (modelos, luego servicios, luego tests).

- **Tamaño**: ~400 líneas cambiadas es una guía de revisión, no un criterio de aceptación. El presupuesto limita el corte, nunca el código: no borres comentarios, docs ni tests ni comprimas código para que quepa.
- **Checklist por commit**: un propósito claro; el repositorio tiene sentido con solo ese commit; rollback razonable; una prueba enfocada con su resultado exacto.
- **Riesgo alto** (`classify.md`): en su propio commit, para que la revisión lo tome solo. El candidato de revisión es el commit, nunca la rama acumulada.
- **Mensaje**: Conventional Commits, un único ámbito. Sin `Co-Authored-By` ni atribución salvo que el repositorio lo pida.
- **Evidencia**: registra en la ficha el commit de cada tarea, el comando de comprobación y su resultado (o `N/A` explícito), y el límite del rollback.
- **Entrega**: estrategia por feature `ask-on-risk` (por defecto), `auto-chain`, `single-pr` o `exception-ok`. Si se parte en varios PR, usa `chained-pr`. Push, PR y merge los decide el usuario.
