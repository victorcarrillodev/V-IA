## Memoria
Si hay herramienta de memoria (Engram):
- **Al empezar**: identifica el proyecto con la herramienta de proyecto actual (nunca inventes un nombre ni busques en todos los proyectos salvo que el usuario lo pida) y recupera por proyecto y tema antes de preguntar al usuario: contexto, búsqueda acotada y solo entonces la observación completa.
- **Qué guardar sin que te lo pidan**: decisiones de arquitectura o diseño, convenciones, bugs resueltos (con causa raíz), descubrimientos no obvios, trampas, cambios de configuración y preferencias del usuario. Una petición con varios arreglos se guarda una vez, al terminar. Autocomprobación tras cada tarea: ¿tomé una decisión, arreglé un bug, aprendí algo no obvio o fijé una convención? Si sí, guarda ANTES de tu respuesta final.
- **Formato**: título (verbo + qué), tipo (bugfix | decision | architecture | discovery | pattern | config | preference), `topic_key` para temas que evolucionan (mismo tema = actualización, léela antes) y contenido con Qué / Por qué / Dónde / Aprendido.
- **Fichas ODD**: el espejo `odd/<feature>/tasks` contiene el documento completo y la ruta del archivo, no un resumen; refréscalo en cada transición de tarea y al cerrar. Si falta la memoria, márcalo pendiente sin bloquear.
- **Entrega**: guardar en memoria nunca cuenta como responder; la respuesta al usuario va completa y al final.
- **Tras una compactación**: guarda el resumen compactado y recupera el contexto antes de continuar.
- Las memorias `needs_review` son contexto obsoleto, no hechos: nunca las marques revisadas sin confirmación del usuario. Sin secretos ni datos personales. La memoria nunca sustituye la verificación: si contradice al código, gana el código y corriges la memoria.
