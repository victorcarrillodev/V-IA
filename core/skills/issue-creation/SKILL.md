---
name: issue-creation
description: Redactar un issue claro y accionable. Úsala cuando el usuario pida crear o redactar un issue o reporte de bug.
---
# Crear un issue

Solo crea el issue en el servicio (por ejemplo con `gh issue create`) si el usuario lo pidió; si no, entrega el texto.

1. Busca antes si ya existe (`gh issue list --search "<palabras>"`); si existe, propón comentar ahí.
2. Si el repositorio tiene formularios o plantillas en `.github/ISSUE_TEMPLATE/`, rellénalos tal cual.
3. Sin plantilla, usa: **Resumen** (una frase) · **Comportamiento actual** · **Comportamiento esperado** · **Pasos para reproducir** (comandos exactos) · **Entorno** (versión, SO) · **Evidencia** (salida literal, sin secretos).
4. Un issue = un problema. Si el reporte mezcla varios, sepáralos.
5. No inventes causas: separa lo observado de lo supuesto y márcalo como hipótesis.
