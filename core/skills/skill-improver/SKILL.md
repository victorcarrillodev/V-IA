---
name: skill-improver
description: Auditar y mejorar una skill existente a partir de fallos observados. Úsala cuando una skill no se activó, se activó mal o el agente la ignoró.
license: "Apache-2.0"
metadata:
  author: "v-ia"
  version: "1.0"
---
# Mejorar una skill

Por defecto **solo audita**: modifica únicamente si el usuario pide aplicar los cambios.

1. **Reproduce** el fallo con la petición real que lo causó. Anota qué esperabas, qué pasó y qué skill debía intervenir.
2. **Diagnostica** en este orden:
   - *No se activó* → la `description` no contiene las palabras del usuario: reescríbela con el disparador concreto ("Úsala cuando…").
   - *Se activó pero no cumplió* → el cuerpo es ambiguo: sustituye consejos por pasos comprobables (sí/no).
   - *Se activó de más* → acótala ("no la uses para…").
3. **Una cosa a la vez** y repite la misma petición; sin reproducción no hay mejora demostrada.
4. **Recorta**, pero nunca borres contenido significativo en silencio: muévelo a `references/` o `assets/` y avísalo. No inventes disparadores ni políticas; marca lo ambiguo para revisión humana.
5. No edites skills gestionadas a mano (se sobrescriben en `v-ia sync`): copia con otro nombre o propón el cambio en el repositorio de V-IA.
6. Informe por severidad (bloqueante / importante / menor) y, en el Log de la ficha: skill, fallo, cambio, resultado de la repetición.
