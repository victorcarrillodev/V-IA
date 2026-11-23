---
name: odd-delegate
description: Decidir si delegar a un subagente y cómo. Úsala antes de lanzar cualquier subagente.
---
# Delegar con motivo

Por defecto trabaja en línea. Delega solo si puedes nombrar uno de estos motivos:

1. **Exploración**: entender requiere más de ~5 búsquedas encadenadas o ~10k tokens; un explorador de solo lectura devuelve un mapa corto con `ruta:línea`.
2. **Escritores paralelos**: dos o más unidades independientes con superficies de edición disjuntas.
3. **Respaldo de contexto**: sesión larga; la siguiente unidad acotada va a un worker nuevo.
4. **Verificador independiente**: cambio de riesgo alto (`v-ia review check` lo marca). Nunca el mismo modelo que lo escribió.

"Muchos archivos" o "tarea grande" no son motivos. Pasa al subagente una **referencia** a `odd/tasks/<feature>.md` y sus `S#`, nunca un parafraseo. Exige: conclusión corta + evidencia + archivos tocados. No amplía el alcance. Anota la delegación en el Log.
