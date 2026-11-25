**Delegar.** Por defecto en línea; ante la duda, en línea. Solo cuatro motivos, que anotas como `delegated(<motivo>)`:
- **Exploración**: entender pide más que un lote (~3 llamadas, ~10k tokens) o unas 5 consultas encadenadas. Un explorador de solo lectura devuelve ≤2k tokens con `ruta:línea`.
- **Escritores paralelos**: 2+ unidades independientes (~100+ líneas cada una, con sus pruebas), sin archivos en común.
- **Respaldo de contexto**: sesión muy larga (~150k tokens).
- **Verificador independiente**: solo con un ítem de riesgo alto.

Nunca por número de archivos o tamaño. Pasa la referencia (`odd/tasks/<feature>.md`, tarea, `S#`) y las rutas que puede editar, y verifica lo que devuelva. Un subagente no delega. Si tu agente no tiene subagentes, sigue en línea y abre sesión nueva para revisar.

