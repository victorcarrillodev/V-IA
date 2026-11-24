---
name: issue-root-resolution
description: Resolver un grupo de issues o fallos relacionados por su causa raíz verificada. Úsala cuando varios reportes parezcan compartir mecanismo o el usuario pida cerrar un clúster.
license: "Apache-2.0"
metadata:
  author: "v-ia"
  version: "1.0"
---
# Resolver por causa raíz

1. Lee los cuerpos y comentarios completos (no solo los títulos) y agrupa por **causa raíz**, no por síntoma.
2. Por cada grupo, un mapa de solo lectura del mecanismo con un ancla `ruta:línea` por afirmación. El mecanismo que cita el reporte es hipótesis; solo el síntoma es evidencia.
3. Ordena las soluciones por lo que **eliminan**: quitar el mecanismo > una guarda estática > un arreglo localizado. Rechaza lo que añade estado, flags o representaciones paralelas.
4. Corte limpio, nunca compatibilidad: no dejes dos caminos "por si acaso".
5. **Cierra solo con evidencia**: pruebas con nombre que fallaban antes y pasan después, ejecutadas contra el binario real. Un hilo con dos modos de fallo no se cierra: se comentan ambos.
6. Audita cada informe de un worker contra el código (es un auto-reporte). Un worktree por escritor.
7. Informa: grupos, causa de cada uno, qué se eliminó, pruebas y delta neto de líneas. Publicar, cerrar o comentar en el host solo a petición.
