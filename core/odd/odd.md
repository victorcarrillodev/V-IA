# ODD — Organic Driven Development

Sigue estos pasos en orden. Cada regla es sí/no.

1. **Autorizar.** Una consulta no es permiso para editar. Si dice "explica", "analiza", "propón" o "sin modificar": NO edites archivos.
2. **Explorar.** Lee el código, las instrucciones del proyecto y las comprobaciones existentes antes de modificar.
3. **Aclarar.** Resuelve dudas investigando. Pregunta al usuario solo si la decisión es suya. Anota tus supuestos.
4. **Clasificar** (ver `classify.md`): Lectura, Acotado o Seguimiento.
5. **Conservar.** En Seguimiento, ejecuta `v-ia mode tracking` y crea (`v-ia task new <feature>`) `odd/tasks/<feature>.md` ANTES de editar: requisitos, criterios, tareas, evidencia, siguiente paso. Guarda también en memoria si hay herramienta.
6. **Implementar y comprobar.** Si hay prueba ejecutable: mírala fallar, implementa, mírala pasar. Si no, di la limitación y haz una comprobación funcional.
7. **Cerrar.** Marca `[x]` solo con una fila en *Evidencia* (comando + resultado observado). Informa qué quedó verificado, qué falló y qué sigue. Al retomar, contrasta documento + memoria + código actual (gana el código).
8. **Entrega.** Commit, push, PR o release SOLO si el usuario lo pide de forma explícita.

Skills disponibles: `odd-resume`, `odd-verify`, `odd-delegate`, `rdd-review`, `work-unit-commits` (índice en `.atl/skill-registry.md` si existe).
Los hooks de V-IA bloquean: editar en modo lectura, editar en seguimiento sin ficha, commit/push/PR sin petición explícita y cerrar con `[x]` sin evidencia. Si un hook te bloquea, corrige la causa; no lo evites.
Comandos de control: `v-ia task check`, `v-ia guard check`, `v-ia review check` (ejecuta build/tests, evalúa riesgo y emite recibo), `v-ia mode`.
