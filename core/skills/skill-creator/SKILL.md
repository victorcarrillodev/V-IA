---
name: skill-creator
description: Crear o mejorar una skill de agente. Úsala cuando el usuario pida una skill nueva o corregir una existente.
---
# Crear una skill

1. Una skill = una carpeta `skills/<nombre>/SKILL.md`. El `name:` del frontmatter debe ser idéntico al nombre de la carpeta (minúsculas y guiones).
2. `description:` en una frase: **qué hace y cuándo usarla**. Es lo único que el agente ve para decidir si la carga, así que debe poder activarse (o no) solo con ella.
3. Cuerpo corto y en pasos numerados con reglas sí/no. Nada de teoría: lo que el agente debe hacer, en orden.
4. Sin duplicar lo que ya cubre otra skill; enlázala por nombre.
5. Nunca incluyas secretos, rutas personales ni órdenes que eludan los controles de seguridad.
6. Para skills de V-IA: añade la carpeta en `core/skills/`, su nombre en `skillNames` (internal/lifecycle) y en la lista de `core/odd/*.md`; `go test ./...` falla si algo queda huérfano.
7. Para mejorar una existente: léela completa, corrige solo lo que falla y mantén su contrato (nombre y propósito).
