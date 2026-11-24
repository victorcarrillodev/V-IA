---
name: skill-registry
description: Descubrir qué skills hay disponibles y cuál aplicar. Úsala al empezar una tarea o cuando dudes si existe una skill para ella.
license: "Apache-2.0"
metadata:
  author: "v-ia"
  version: "1.0"
---
# Registro de skills

1. Lee `.atl/skill-registry.md` si existe: es un **índice** (nombre, descripción, ruta, ámbito), no un resumen ni un compilador de reglas. Si no existe o está viejo: `v-ia skill-registry refresh`. Nunca lo sustituyas por una copia de memoria o de sesión.
2. Elige por la `description`, no por el nombre. Si dos encajan, la más específica; si ninguna, trabaja sin skill. El proyecto gana al usuario con el mismo nombre; entre las del usuario, la primera en el orden de escaneo.
3. **Carga solo la que vas a usar** (su `SKILL.md` completo). No cargues el resto "por si acaso".
4. Al delegar, pasa a los subagentes la **ruta exacta** de las skills relevantes (máximo ~5), no un resumen, y pídeles que informen si recibieron las rutas (`paths-injected`) o tuvieron que releer el registro.
5. Tras crear, editar o borrar una skill, `v-ia skill-registry refresh`. Añade `.atl/` al `.gitignore`.
6. Informa: ruta del registro, nº de skills, si cambió y duplicadas.
