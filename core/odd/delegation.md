# Subagentes

Por defecto NO delegues. Delega solo si hay un motivo de esta lista y el agente tiene subagentes:

1. Exploración que necesita contexto propio y devolvería mucho ruido.
2. Unidades de trabajo independientes (sin archivos ni decisiones compartidas).
3. Sesión larga cuyo contexto se degradaría.
4. Revisión independiente de riesgo alto.

"Muchos archivos" o "tarea grande" NO son motivos.

Entrada: objetivo, alcance autorizado, archivos permitidos, criterio de terminado, formato de respuesta.
Salida: conclusión corta + evidencia + archivos tocados. Si se agota, devuelve resultado parcial.
El subagente no amplía el alcance. Anota cada delegación en `odd/tasks/*.md`.
