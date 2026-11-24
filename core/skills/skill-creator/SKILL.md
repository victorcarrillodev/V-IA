---
name: skill-creator
description: Crear o rediseñar una skill (SKILL.md) con frontmatter correcto y descripción activable. Úsala cuando el usuario pida una skill nueva.
license: "Apache-2.0"
metadata:
  author: "v-ia"
  version: "1.0"
---
# Crear una skill

No crees una skill si el patrón es trivial o de una sola vez: va a las instrucciones del proyecto.

1. **Frontmatter**: `name` (minúsculas con guiones, igual que la carpeta) y `description` en UNA línea: primero el disparador ("Úsala cuando…"), luego qué hace; ~160 caracteres recomendado, 250 máximo. Es lo único que el agente ve para decidir si la carga.
2. **Estructura del cuerpo**: contrato de activación, reglas duras, puertas de decisión (tabla si hay varias), pasos de ejecución numerados y comprobables (sí/no), y contrato de salida (qué informa al terminar).
3. **Presupuesto**: cuerpo de ~180-450 tokens (700 recomendado, 1000 máximo). Lo que sobre va a `references/` o `assets/` y se enlaza con rutas locales. Cada línea que no cambie la conducta, fuera. No añadas una sección de palabras clave.
4. Reglas de V-IA: las skills gestionadas (`<!-- v-ia:managed-file -->`) se sobrescriben en `v-ia sync`; crea la tuya con otro nombre. Pruébala con la petición real que debía activarla; si no se activa, reescribe la `description`.
5. Tras crearla, `v-ia skill-registry refresh`.
