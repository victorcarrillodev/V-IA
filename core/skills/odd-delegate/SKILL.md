---
name: odd-delegate
description: Decidir si delegar a un subagente y cómo. Úsala antes de lanzar cualquier subagente o escritor paralelo.
license: "Apache-2.0"
metadata:
  author: "v-ia"
  version: "1.0"
---
# Delegar con motivo

Por defecto trabaja en línea. Ningún disparador se cumple "por defecto": ante la duda, en línea. Delegar sin motivo es un defecto de enrutado aunque el trabajo salga bien.

## Motivos (los únicos)
1. **Exploración**: entender excede el lote en línea (~3 llamadas, ~10k tokens) o requiere más de ~5 consultas encadenadas → `v-ia-explorer` (solo lectura). Entrega un mapa de ~2k tokens con `ruta:línea`; compruébalo con una muestra.
2. **Escritores paralelos**: dos o más unidades independientes de comportamiento (código + pruebas + docs juntos), con superficies de edición disjuntas y varios minutos de trabajo cada una (~100+ líneas), lanzadas juntas en un mismo turno. Unidades de ~100 líneas o menos van en línea. Nunca separes en un worker de tests, otro de docs y otro de código.
3. **Respaldo de contexto**: la sesión es larga (~150k tokens, orientativo) y la siguiente unidad acotada va a un worker nuevo.
4. **Verificador independiente**: solo si la línea `Risk:` nombra un ítem de riesgo alto (`classify.md`), no por tamaño ni por un tier. Nunca el modelo que escribió el cambio.

"Muchos archivos", "tarea grande" o el precio del modelo nunca son motivos. Si eres un subagente, no delegues.

## Contrato del escritor
- Pasa una **referencia** (`Spec: odd/tasks/<feature>.md, T2, S3-S4`): el worker lee hasta `## Log` y reporta qué `S#` cubrió. Sin ficha, la petición literal.
- Incluye `## Allowed edit surfaces`: rutas exactas o globs estrechos, uno por línea; nunca `.`, la raíz ni rutas absolutas. Fuera de eso se niega a escribir.
- Misión autocontenida: objetivo en una frase, rutas, restricciones, evidencia esperada, rutas exactas de los `SKILL.md` relevantes (no un resumen) y fallos conocidos de la base.
- Paralelo: el padre es dueño de git; los writers no ejecutan formatters, generadores ni installs globales. Otro repositorio va en un worktree nuevo. Cada prompt nombra los helpers existentes que debe reutilizar.
- Al volver: revisa el diff contra su superficie; el reporte del worker es un auto-reporte, verifícalo. Al final una prueba de costura (suite completa, una vez).

## Verificación por riesgo
- **passive**: relectura estructural. **medium**: autoverificación del writer. **high** (ítem nombrado): autoverificación + verificador independiente de solo lectura.
- Un lote de corrección que arregle todos los bloqueadores y un único recheck limitado a ellos; una segunda corrección solo si el MISMO bloqueador sigue fallando. Si quedan abiertos, la unidad pasa a `partial` y decides tú o el usuario; no repitas el mismo intento.

Anota ruta (`inline` | `delegated(<motivo>)`) y evidencia en la ficha.
