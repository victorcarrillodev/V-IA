# Manual de V-IA

Referencia completa. Para una visión rápida e instalación, ver el [README](../README.md).

Contenido:

1. [Arquitectura](#arquitectura)
2. [Instalación y ciclo de vida](#instalación-y-ciclo-de-vida)
3. [Los hooks, uno por uno](#los-hooks-uno-por-uno)
4. [Modos: lectura, edición y seguimiento](#modos-lectura-edición-y-seguimiento)
5. [La ficha ODD (`odd/tasks/`)](#la-ficha-odd-oddtasks)
6. [RDD: recibo, riesgo y revisor](#rdd-recibo-riesgo-y-revisor)
7. [Skills y registro](#skills-y-registro)
8. [Memoria (Engram)](#memoria-engram)
9. [Seguridad](#seguridad)
10. [Agentes soportados](#agentes-soportados)
11. [Referencia de comandos](#referencia-de-comandos)
12. [Dónde guarda cosas V-IA](#dónde-guarda-cosas-v-ia)
13. [Variables de entorno](#variables-de-entorno)
14. [Solución de problemas](#solución-de-problemas)
15. [Estructura del código](#estructura-del-código)
16. [Cómo ampliarlo](#cómo-ampliarlo)

---

## Arquitectura

```
core/ (fuente de verdad, embebida en el binario)
  odd/odd.md          reglas ODD (comunes)
  odd/enforced.md     cola del bloque para agentes CON hooks (Claude Code)
  odd/advisory.md     cola del bloque para agentes SIN hooks
  odd/classify.md     reglas de clasificación Lectura/Acotado/Seguimiento
  odd/templates/task.md   plantilla de la ficha
  memory/memory-protocol.md   protocolo de memoria
  skills/*/SKILL.md   5 skills
  agents/v-ia-reviewer.md   subagente revisor
        │
        ▼  lifecycle.Install / Sync / Uninstall  (solo PLANIFICA)
  Plan = lista de Acciones (create / add / update / unchanged / conflict / delete)
        │
        ▼  core.Apply  (backup → escritura atómica → rollback si falla)
  ~/.claude/…  ~/.config/opencode/…  ~/.codex/…  …
```

Principios:

- **Una fuente de verdad, muchas vistas.** Las reglas viven en `core/`; cada agente recibe su versión renderizada.
- **Seguro por defecto.** `--dry-run` primero; backup antes de cada escritura; idempotente; nunca pisa lo que editaste; `restore` garantizado.
- **Degradación elegante.** Sin hooks, ODD sigue funcionando como instrucciones; los hooks añaden el bloqueo.
- **Los hooks fallan abiertos.** Si V-IA no entiende la entrada de un hook, no bloquea al agente; solo bloquea cuando una regla se incumple.
- **Estado fuera del modelo.** La clasificación (lectura/edición/seguimiento) se guarda en `.v-ia/state.json`; el agente no decide si está restringido.

### Bloques gestionados

Cada archivo de instrucciones recibe:

```
<!-- v-ia:managed begin id=odd version=0.3.0 sha256=… -->
…contenido…
<!-- v-ia:managed end id=odd -->
```

Solo se reescribe lo que hay entre marcadores. El `sha256` detecta si lo editaste a mano: en ese caso es un **conflicto** y no se pisa sin `--force`.

Los archivos completos (skills, subagente) llevan al final `<!-- v-ia:managed-file sha256=… -->`. Un archivo sin marca, o con la marca pero editado, **es tuyo** y no se toca ni se borra.

---

## Instalación y ciclo de vida

### `v-ia install`

```
v-ia install [--dry-run] [--yes] [--force] [--agent a,b] [--no-hooks] [--no-permissions] [--engram]
```

| Flag | Efecto |
|---|---|
| `--dry-run` | Muestra el diff; no escribe nada. |
| `--yes` | Aplica. Sin `--yes` ni `--dry-run`, se niega a escribir. |
| `--force` | Sobrescribe lo editado a mano (con backup). |
| `--agent` | Lista separada por comas. Por defecto `claude-code`. Válidos: `claude-code, opencode, codex, gemini-cli, kilocode, qwen-code, windsurf, generic`. |
| `--no-hooks` | No instala hooks. Con `--no-permissions` tampoco toca `settings.json`. |
| `--no-permissions` | No siembra la deny-list. |
| `--engram` | Tras instalar, ejecuta `engram setup claude-code` (requiere `engram`). |

- Es **idempotente**: una segunda ejecución dice "nada que cambiar".
- Antes de escribir crea un backup en `~/.v-ia/backups/<fecha>/` con un `manifest.json`.
- Si una escritura falla a mitad, **hace rollback** automático.

Con `--agent generic` escribe `AGENTS.md` en el **directorio actual** (útil para cualquier agente que lea ese archivo).

### `v-ia` sin argumentos (TUI)

En un terminal interactivo abre un asistente (Bubble Tea): bienvenida → agentes → componentes → revisión del diff → aplicar → resultado. Hace lo mismo que `install`, con hooks y permisos activados. Sin terminal interactivo, falla y te lista los comandos.

### `v-ia sync`

Actualiza **solo lo que ya está instalado**. No crea componentes nuevos ni recrea lo que borraste. Úsalo tras `upgrade` o tras cambiar de versión. Mismos flags que `install`.

### `v-ia uninstall`

Quita los bloques ODD, los hooks, las skills y el subagente. Conserva:
- todo lo que hay fuera de los marcadores;
- los archivos gestionados que editaste a mano;
- la **deny-list** de permisos (es decisión tuya; quítala editando `permissions.deny`).

Si un archivo solo contenía nuestro bloque, se borra.

### `v-ia upgrade`

```
v-ia upgrade --src <ruta-al-repo>      # o V_IA_SRC=<ruta>
```
Hace `git pull --ff-only` en el repo, recompila (`go build`) y reemplaza el binario en uso. Después ejecuta `v-ia sync`. No hay releases publicados: se actualiza desde el código fuente.

### `v-ia restore`

```
v-ia restore --list              # backups, del más antiguo al más reciente
v-ia restore latest --yes        # vuelve al estado previo al último cambio
```
Restaura también los archivos que `uninstall` borró.

### `v-ia doctor`

Diagnóstico de solo lectura:
- si `claude` está en el PATH;
- si cada archivo de instrucciones tiene el bloque al día, pendiente o editado a mano;
- si los hooks están instalados;
- qué agentes detectados se hacen cumplir por bloqueo y cuáles son solo recomendación;
- si `engram` está disponible.

Devuelve código de salida 1 si hay algún `✗`.

---

## Los hooks, uno por uno

Instalados en `~/.claude/settings.json`. Todos ejecutan `v-ia hook <pre|prompt|stop|session>` y leen el JSON que entrega Claude Code por stdin. **Código de salida 2 = bloquear** (el mensaje va al agente).

### `UserPromptSubmit` — clasifica cada mensaje tuyo

Decide el modo y si autorizaste entregar, **solo con tu mensaje**:

- **Lectura** si dice `sin modificar/editar/tocar/cambiar`, `no modifiques/edites/toques/cambies`, `solo lectura`, `read-only` — **esto gana siempre**.
- **Lectura** también si dice `explica…`, `analiza…`, `propón…`, `qué hace`, `cómo funciona` **y no** contiene un verbo de edición (`corrige, arregla, implementa, crea, añade, cambia, elimina, escribe, refactoriza, actualiza, modifica, edita, haz…`).
- **Edición** en cualquier otro caso. (Si estabas en `tracking`, se mantiene.)
- **Entrega autorizada** si dice `commit, push, pull request, release, publica, sube, pr`, salvo que lo niegue (`sin commit`, `no hagas push`…). **La autorización dura solo ese mensaje.**

En modo lectura guarda una huella del árbol (`guard start`).

### `PreToolUse` — decide si una acción se permite

| Acción | Se bloquea cuando |
|---|---|
| `Edit`, `Write`, `MultiEdit`, `NotebookEdit` | Modo lectura; o modo seguimiento sin ninguna ficha en `odd/tasks/` (crear la ficha sí se permite). |
| `Bash` con comando destructivo | Siempre: `rm -rf` de `/`, `~`, `$HOME`, `.`, `..`; `git push --force/-f`; `git reset --hard`; `git clean -f`; `chmod -R 777`. |
| `Bash` con `git commit/push/tag`, `gh pr create/merge`, `gh release create` | No lo pediste en tu último mensaje; **o** RDD está activo, el proyecto tiene comprobaciones y no hay recibo válido; **o** el riesgo es alto y falta el visto bueno del revisor. |

### `Stop` — comprueba antes de dejar cerrar

Bloquea el cierre (una vez por turno, para evitar bucles) si:
- estás en modo lectura y **hay archivos cambiados** (lista cuáles; así se detecta un `sed -i` desde Bash);
- alguna ficha de `odd/tasks/` no valida (`task check`).

### `SessionStart` — contexto al abrir sesión

- Si `engram` está en el PATH, ejecuta `engram context` (máx. 5 s) e inyecta hasta 3000 caracteres, recordando que **si la memoria contradice al código, gana el código**.
- Si el proyecto tiene `.atl/skill-registry.md`, lo refresca y le dice al agente que lo lea.

---

## Modos: lectura, edición y seguimiento

| Modo | Cómo se activa | Qué impide |
|---|---|---|
| `read` | Automático por tu mensaje (ver arriba), o `v-ia mode read` | Cualquier edición. |
| `edit` | Automático por defecto, o `v-ia mode edit` | Nada especial. |
| `tracking` | **Manual**: `v-ia mode tracking` | Editar sin ficha `odd/tasks/*.md`. Persiste entre mensajes hasta `v-ia mode edit`. |

`v-ia mode` (o `mode status`) muestra el modo actual. El estado vive en `.v-ia/state.json` de cada proyecto.

**Cuándo usar Seguimiento** (regla de `classify.md`): trabajo de varias sesiones, requisitos que hay que conservar, integraciones, migraciones o riesgo alto. No se decide por número de archivos.

---

## La ficha ODD (`odd/tasks/`)

Se crea con:

```bash
v-ia task new "nombre de la feature" --request "tu petición, literal"
```

Genera `odd/tasks/<slug>.md`:

```markdown
---
feature: nombre de la feature
status: draft
created: 2026-11-23
updated: 2026-11-23
authorized-by: <usuario / fecha / alcance>
delivery: ask-on-risk
test-runner: <comando>
---
## Specs
- S1: tu petición literal (textos, mensajes de error y ejemplos exactos)

## Tareas
- [ ] T-1 (S1) route=inline commit=- descripción

## Evidencia
| Tarea | Comando / comprobación | Resultado observado | Fecha |
|---|---|---|---|

## Log
- L1: tu petición original, literal
```

Reglas que valida `v-ia task check` (y el hook `Stop`):

- Debe existir `## Specs` con al menos un `S#`, y sin el texto de relleno `<pega…`.
- Debe existir `## Log` con `L1` no vacío.
- Cada tarea debe citar un `S#` que exista.
- Cada tarea `[x]` necesita una fila en **Evidencia** con comando **y** resultado no vacíos (`| T-1 | go test ./... | ok | hoy |`).
- Debe tener frontmatter (`---`).

Sin `--request`, la ficha se crea con relleno y **no valida** hasta que pegues tu petición literal. Los subagentes deben recibir una **referencia** a la ficha y sus `S#`, nunca un resumen.

`v-ia task status` lista cada ficha con su estado y avance (`2/5 tareas`).

> `task check` valida la **forma** de la evidencia, no que el comando se haya ejecutado de verdad.

---

## RDD: recibo, riesgo y revisor

RDD (Receipt-Driven Development) liga una revisión a la **versión exacta** del código.

### `v-ia review check`

1. Ejecuta las comprobaciones del proyecto (máx. 10 min en total por comando):
   - `.v-ia/checks.txt` si existe (un comando por línea; `#` para comentarios) — **manda sobre lo demás**;
   - si no: `go.mod` → `go build ./...`, `go vet ./...`, `go test ./...`; `package.json` con `"test"` → `npm test --silent`.
   - Sin comprobaciones detectadas **falla**: un recibo vacío no demuestra nada.
2. Calcula el `candidate_id` (hash de todo el árbol, sin `.git`, `.v-ia`, `node_modules`, `.serena`).
3. Evalúa el riesgo.
4. Guarda `.v-ia/receipt.json` y devuelve `0` solo si todo pasó.

**Si cambia un solo byte del árbol, el recibo y cualquier visto bueno quedan invalidados.**

### Riesgo (`v-ia review assess [--base-ref REF]`)

Compara con `HEAD` (o el árbol vacío si no hay commits) y clasifica:

| Nivel | Cuándo |
|---|---|
| `high` | Ruta sensible (`auth`, `security`, `crypto`, `secret`, `token`, `password`, `permission`, `migration`, `schema`, `.sql`, `.github/workflows/`, `Dockerfile`, `release`/`deploy`) **o** contenido añadido sensible (`DROP/DELETE FROM/ALTER TABLE/TRUNCATE`, claves/secretos asignados, `sync.Mutex`, `go func`, `atomic.`, `chmod -R`, `exec.Command`). |
| `medium` | ≥150 líneas o ≥5 archivos de código; o código sin tests y >30 líneas. |
| `passive` | Solo docs/tests, sin cambios, o cambio pequeño y acotado. |
| `unknown` | No se pudo evaluar. **Nunca se asume riesgo bajo.** |

Solo `high` marca `review_due`.

### Revisor independiente

Con riesgo alto, el agente debe lanzar el subagente **`v-ia-reviewer`** (solo lectura): revisa `git diff HEAD` con cuatro lentes (**R**isk, **R**esilience, **R**eadability, **R**eliability), exige un caso concreto por hallazgo y registra:

```bash
v-ia review ack --verdict approve --reviewer v-ia-reviewer
v-ia review ack --verdict changes --reviewer v-ia-reviewer --note "resumen"
```
El ack solo se acepta si el árbol sigue siendo el del recibo. `changes` no es aprobación. Tras una corrección hay que repetir `review check`.

### Modo ON/OFF

```bash
v-ia review mode status
v-ia review mode disable [--scope project|global]    # por defecto: project
v-ia review mode enable  [--scope project|global]
```
RDD está **ON por defecto**. Un OFF (de proyecto o global) lo apaga; apagado no se exige recibo ni revisor (las comprobaciones ordinarias siguen existiendo). El global vive en `~/.v-ia/config.json`; el de proyecto en `.v-ia/review-mode`.

### `v-ia review verify`

Comprueba que hay un recibo `pass` para el árbol actual (y, con RDD activo y riesgo alto, el visto bueno).

> Un recibo `pass` **nunca** autoriza commit/push/PR. Eso solo lo hace tu mensaje.

---

## Skills y registro

Instaladas en `~/.claude/skills/` (solo Claude Code):

| Skill | Para qué |
|---|---|
| `odd-resume` | Retomar una feature: leer la ficha, contrastar con el código (gana el código), reabrir solo lo invalidado. |
| `odd-verify` | Cerrar con evidencia: ejecutar comprobaciones, rellenar Evidencia, `task check` + `review check`. |
| `odd-delegate` | Decidir si delegar. Solo con un motivo: exploración, escritores paralelos independientes, respaldo de contexto o verificador independiente. "Muchos archivos" no es motivo. |
| `rdd-review` | Flujo de revisión según el riesgo. |
| `work-unit-commits` | Commits por unidad de trabajo (~400 líneas como guía, Conventional Commits, riesgo alto en su propio commit). |

### Registro de skills

```bash
v-ia skill-registry list       # nombre, alcance, descripción
v-ia skill-registry refresh    # escribe .atl/skill-registry.md (no reescribe si no cambió)
```
Escanea `~/.claude/skills/` (global) y `<proyecto>/.claude/skills/` (proyecto); **proyecto gana a global** si el nombre se repite. Si el proyecto ya tiene `.atl/skill-registry.md`, el hook `SessionStart` lo mantiene al día.

---

## Memoria (Engram)

V-IA **no implementa memoria propia**: delega en [Engram](https://github.com/Gentleman-Programming/engram).

- El bloque ODD incluye el protocolo: recuperar por proyecto antes de preguntar; guardar tras decisiones, descubrimientos, bugs resueltos y convenciones (sin secretos); **la memoria nunca sustituye la verificación**.
- `v-ia install --engram` ejecuta `engram setup claude-code`.
- El hook `SessionStart` inyecta `engram context` al abrir sesión.

Sin `engram` en el PATH, todo lo demás funciona; solo no hay memoria entre sesiones.

---

## Seguridad

**Deny-list** que se siembra en `permissions.deny` de `settings.json` (solo si esa clave no existe; una lista vacía tuya se respeta):

`Read(~/.ssh/**)`, `Read(**/.env)`, `Read(**/.env.*)`, `Read(**/*.pem)`, `Read(**/*.key)`, `Read(**/*.p12)`, `Read(**/*.pfx)`, `Read(~/.aws/credentials)`, `Read(~/.config/gh/hosts.yml)`, `Read(**/secrets/**)`, `Read(~/.credentials/**)`.

**Comandos bloqueados siempre** por el hook: ver tabla de `PreToolUse`.

**Lo que no cubre:** leer un secreto con `cat` desde Bash no se bloquea; los comandos destructivos con envoltorios inusuales (`env`, `xargs`, shells anidados) pueden escaparse; no hay sandbox.

**Privacidad:** V-IA no envía nada a ningún servicio. El único proceso externo que lanza es `engram` (y solo si existe).

---

## Agentes soportados

| Agente | Archivo que escribe | Cumplimiento | Verificado |
|---|---|---|---|
| claude-code | `~/.claude/CLAUDE.md` + `settings.json` + skills + subagente | **Hooks (bloqueo)** | E1, E2, E3, E7 con Claude real |
| opencode | `$XDG_CONFIG_HOME/opencode/AGENTS.md` (por defecto `~/.config/opencode/`) | Instrucciones | E1–E4 con OpenCode 1.19.5 |
| codex | `$CODEX_HOME/AGENTS.md` (por defecto `~/.codex/`) | Instrucciones | No |
| gemini-cli | `~/.gemini/GEMINI.md` | Instrucciones | No |
| kilocode | `$XDG_CONFIG_HOME/kilo/AGENTS.md` | Instrucciones | No |
| qwen-code | `~/.qwen/QWEN.md` | Instrucciones | No |
| windsurf | `~/.codeium/windsurf/memories/global_rules.md` | Instrucciones | No |
| generic | `AGENTS.md` del directorio actual | Instrucciones | No |

Los agentes **sin hooks** reciben un bloque distinto que no promete bloqueos ni skills, y que les pide ejecutar por su cuenta `v-ia guard start/check`, `task check` y `review check`.

No soportados todavía: Cursor, VS Code Copilot, Kiro, Kimi, Antigravity y el resto de los 17 de Gentle-AI.

---

## Referencia de comandos

| Comando | Qué hace |
|---|---|
| `v-ia` | TUI de instalación (requiere terminal interactivo). |
| `v-ia version` | Muestra la versión. |
| `v-ia install [flags]` | Instala/actualiza bloques, hooks, skills, subagente (ver flags arriba). |
| `v-ia sync [flags]` | Actualiza solo lo ya instalado. |
| `v-ia uninstall [flags]` | Retira lo instalado (conserva lo editado a mano). |
| `v-ia upgrade --src DIR` | Recompila desde el repo y reemplaza el binario. |
| `v-ia restore --list` / `restore latest --yes` | Backups y restauración. |
| `v-ia doctor` | Diagnóstico de solo lectura. |
| `v-ia mode [status\|read\|edit\|tracking]` | Modo del proyecto actual. |
| `v-ia task new <feature> [--request "…"]` | Crea la ficha en `odd/tasks/`. |
| `v-ia task check` | Valida todas las fichas. |
| `v-ia task status` | Estado y avance de cada ficha. |
| `v-ia guard start` / `guard check` | Huella del árbol y diff (detecta cambios en una consulta de solo lectura). |
| `v-ia review check` | Comprobaciones + riesgo + recibo. |
| `v-ia review verify` | Valida el recibo actual. |
| `v-ia review assess [--base-ref REF]` | Evalúa el riesgo (JSON). |
| `v-ia review ack --verdict approve\|changes [--reviewer X] [--note "…"]` | Registra el veredicto del revisor. |
| `v-ia review mode status\|enable\|disable [--scope global\|project]` | Interruptor RDD. |
| `v-ia skill-registry list\|refresh` | Índice de skills. |
| `v-ia hook pre\|prompt\|stop\|session` | **Interno**: lo invocan los hooks; no lo ejecutes a mano. |

No existe `--help` general: esta tabla es la referencia. Códigos de salida: `0` bien, `1` fallo o regla incumplida, `2` uso incorrecto (y, en `hook`, "bloquear").

---

## Dónde guarda cosas V-IA

**Global (`~/.v-ia/`)**

| Ruta | Contenido |
|---|---|
| `~/.v-ia/backups/<fecha>/` | `manifest.json` + copia de cada archivo antes de tocarlo. |
| `~/.v-ia/config.json` | Interruptor global de RDD. |

**Por proyecto** (conviene añadir `.v-ia/` y `.atl/` a tu `.gitignore`; `odd/tasks/` sí se versiona)

| Ruta | Contenido |
|---|---|
| `.v-ia/state.json` | Modo (`read/edit/tracking`) y si hay entrega autorizada. |
| `.v-ia/guard.json` | Huella del árbol para `guard check`. |
| `.v-ia/receipt.json` | Último recibo: comprobaciones, candidato, riesgo y veredicto del revisor. |
| `.v-ia/checks.txt` | (opcional, tuyo) comprobaciones de `review check`. |
| `.v-ia/review-mode` | (opcional) `off` para apagar RDD en este proyecto. |
| `.atl/skill-registry.md` | Índice de skills. |
| `odd/tasks/*.md` | Fichas de seguimiento. |

---

## Variables de entorno

| Variable | Efecto |
|---|---|
| `V_IA_HOME` | Sustituye a `$HOME`. **Úsala para probar sin tocar tu configuración real.** |
| `XDG_CONFIG_HOME` | Cambia dónde se buscan OpenCode y Kilo. |
| `CODEX_HOME` | Cambia dónde se busca la configuración de Codex. |
| `V_IA_SRC` | Repo por defecto de `upgrade`. |
| `V_IA_BIN` | (scripts e2e) ruta del binario a probar. |
| `OC_MODEL` | (script e2e de OpenCode) modelo a usar. |

Prueba segura:
```bash
export V_IA_HOME=$(mktemp -d)
v-ia install --yes && v-ia doctor     # todo ocurre dentro del directorio temporal
```

---

## Solución de problemas

**`v-ia: command not found`** — `~/.local/bin` no está en tu `PATH`.

**Los hooks no hacen nada** — `v-ia doctor`; si dice "hooks ODD no instalados", ejecuta `v-ia install --yes`. Comprueba además que `which v-ia` apunta al binario (los hooks guardan su ruta absoluta; si lo mueves, repite `install`).

**Bloquea una edición que sí querías** — tu mensaje se clasificó como lectura (`explica`, `sin modificar`…). Reformula con un verbo de edición ("corrige…") o ejecuta `v-ia mode edit`.

**"trabajo de seguimiento sin ficha"** — estás en `tracking`: crea la ficha con `v-ia task new …` o vuelve con `v-ia mode edit`.

**No deja hacer commit** — mira el mensaje:
- "requieren que el usuario lo pida" → dilo en tu mensaje ("haz commit");
- "sin recibo válido" → `v-ia review check`;
- "falta la revisión independiente" → riesgo alto: lanza el subagente `v-ia-reviewer`.

**"no hay comprobaciones"** — crea `.v-ia/checks.txt`, p. ej.:
```
# una orden por línea
make test
./scripts/lint.sh
```

**Quiero desactivar RDD en un proyecto** — `v-ia review mode disable`.

**"el bloque gestionado fue editado a mano"** — editaste entre los marcadores. Mueve tu texto fuera del bloque y repite, o usa `--force` (hace backup).

**`settings.json no es JSON válido`** — V-IA se niega a tocarlo. Corrígelo (sin comentarios ni comas finales) y repite.

**Quiero volver atrás** — `v-ia restore latest --yes` (último cambio) o `v-ia uninstall --yes`.

**El hook `Stop` bloquea y no sé por qué** — el mensaje lista el archivo y la línea. Normalmente: una ficha sin completar o una tarea `[x]` sin evidencia.

---

## Estructura del código

```
cmd/v-ia/            CLI: parseo de comandos y salida
core/                archivos embebidos (reglas, plantilla, skills, subagente, memoria)
internal/core/       bloques gestionados, diff, plan, apply/restore (sin UI)
internal/lifecycle/  planes de install, sync y uninstall
internal/adapters/   tabla de agentes, rutas y bloque por agente
internal/settings/   merge de hooks y deny-list en settings.json
internal/hook/       reglas de los hooks (clasificar, bloquear, session)
internal/task/       creación y validación de fichas
internal/guard/      huella del árbol (guard) y candidate_id
internal/review/     checks, riesgo, recibo, ack, modo ON/OFF
internal/registry/   índice de skills
internal/doctor/     diagnóstico
internal/tui/        asistente Bubble Tea
scripts/             e2e contra Claude Code y OpenCode reales
```

## Cómo ampliarlo

**Añadir un agente**: añade una entrada a `adapters.All` (`Name`, `Binary`, `Hooks`, `Targets`), un caso en `adapters_test.go` y documenta si la ruta está verificada. Solo ponle `Hooks: true` si V-IA instala hooks de bloqueo para él (hoy solo Claude Code).

**Cambiar las reglas ODD**: edita `core/odd/*.md` (se embeben al compilar), sube `Version` en `internal/adapters/adapters.go` y ejecuta `v-ia sync`.

**Añadir una skill**: crea `core/skills/<nombre>/SKILL.md` con frontmatter (`name`, `description`) y añade el nombre a `skillNames` en `internal/lifecycle/lifecycle.go`.

**Añadir una regla de bloqueo**: `internal/hook/hook.go` (`preTool`, `dangerRe`) con un test en `hook_test.go`.

Antes de subir cambios: `go build ./... && go vet ./... && go test ./...` y, si tocaste hooks o instrucciones, los scripts `e2e`.
