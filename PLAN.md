# V-IA — Plan: ODD universal para cualquier agente de IA

> Objetivo: un sistema propio y global en el PC que haga que **cualquier agente de IA** (Claude Code, OpenCode, Codex, Gemini CLI, Cursor, Copilot, etc.) trabaje con **ODD (Organic Driven Development)**, con memoria (Engram), skills con registro, revisión RDD y subagentes con motivo concreto.
>
> Referencia conceptual: guía no oficial de Gentle-AI v4.6.0 (HieloCode) y repo `Gentleman-Programming/gentle-ai`. Lo que no esté confirmado en esa guía se marca **[SUPUESTO]** y se verifica en la fase indicada.

---

## 0. Principios de diseño

1. **Una sola fuente de verdad, muchos adaptadores.** ODD vive en archivos de texto neutrales; cada agente recibe una *vista renderizada* en el formato que entiende. Nada de lógica atada a un agente.
2. **Degradación elegante.** Si un agente no soporta hooks, MCP o subagentes, ODD sigue funcionando solo con instrucciones + archivos. Las capacidades extra son aditivas.
3. **Texto plano y auditable.** Todo estado vive en archivos legibles (`odd/tasks/*.md`, `.atl/skill-registry.md`, `.engram/`). Cualquier IA (o humano) puede leerlos sin herramientas especiales.
4. **Seguro por defecto.** Dry-run antes de escribir, backup antes de modificar, idempotencia, nunca pisar ediciones manuales del usuario, restore garantizado.
5. **Proporcionalidad.** El proceso se adapta al trabajo: preguntas y cambios pequeños no generan burocracia.
6. **Evidencia sobre afirmaciones.** Nada se marca "hecho" sin resultado observado.
7. **Privacidad.** Memoria local por defecto; nada se sube a la nube sin acción explícita.

---

## 1. Alcance y no-alcance

**Incluye**
- Núcleo ODD (instrucciones, plantilla de seguimiento, reglas de clasificación).
- Memoria persistente tipo Engram (integración, y fallback propio si no está).
- Skills con registro e índice.
- RDD (revisión con "recibos" vinculados a una versión exacta del cambio).
- Política de subagentes.
- Instalador/CLI `v-ia` con `install`, `sync`, `doctor`, `restore`, `status`.
- Adaptadores por agente + matriz de capacidades.

**No incluye (por ahora)**
- Un agente propio ni un modelo propio (V-IA *configura* agentes existentes, igual que Gentle-AI).
- Servicio cloud ni sincronización remota de memoria.
- Autorización automática de commit/push/PR (siempre decide el usuario).

---

## 2. Arquitectura

```
V-IA/
├── core/                       # FUENTE DE VERDAD (agnóstica del agente)
│   ├── odd/
│   │   ├── odd.md              # instrucciones ODD canónicas
│   │   ├── classify.md         # reglas de clasificación (lectura/acotado/seguimiento)
│   │   └── templates/
│   │       └── task.md         # plantilla odd/tasks/<feature>.md
│   ├── memory/
│   │   ├── memory-protocol.md  # cuándo/qué guardar y recuperar
│   │   └── mcp/                # config MCP de Engram / servidor propio
│   ├── skills/
│   │   ├── odd-explore/SKILL.md
│   │   ├── odd-resume/SKILL.md
│   │   ├── odd-verify/SKILL.md
│   │   ├── odd-delegate/SKILL.md
│   │   ├── rdd-review/SKILL.md
│   │   ├── skill-registry/SKILL.md
│   │   └── skill-creator/SKILL.md
│   └── rdd/
│       ├── rdd.md              # protocolo de revisión
│       └── receipt.schema.json # formato del recibo
├── adapters/                   # un directorio por agente
│   ├── claude-code/
│   ├── opencode/
│   ├── codex/
│   ├── gemini-cli/
│   ├── cursor/
│   ├── copilot/
│   └── generic/                # AGENTS.md genérico para cualquier otro
├── cli/                        # instalador `v-ia`
├── tests/                      # tests del instalador y de los adaptadores
└── docs/
```

### Flujo de render

```
core/*.md  ──(render + plantillas por agente)──►  destino global del agente
                                                   ~/.claude/…  ~/.config/opencode/…
                                                   ~/.codex/AGENTS.md  ~/.gemini/GEMINI.md …
```

Cada archivo escrito lleva un **marcador de gestión** para que `sync` sepa qué es suyo y qué editó el usuario:

```
<!-- v-ia:managed begin id=odd version=1.0.0 sha256=… -->
…contenido…
<!-- v-ia:managed end -->
```

Solo se reescribe el bloque entre marcadores; lo de fuera nunca se toca.

---

## 3. Núcleo ODD

### 3.1 Instrucciones canónicas (`odd.md`)

Contenido (≈1 página, imperativo, sin ambigüedad):

1. **Autorizar.** Distinguir consulta de petición de cambio. Investigar/proponer ≠ permiso para editar.
2. **Explorar.** Leer código, instrucciones del proyecto y comprobaciones existentes antes de modificar.
3. **Aclarar.** Resolver dudas concretas investigando; preguntar al usuario solo si la decisión le pertenece. Registrar supuestos.
4. **Clasificar** por intención/riesgo/recuperabilidad (no por nº de archivos):
   - **Lectura** → responder sin editar.
   - **Acotado** → explorar, implementar, comprobar.
   - **Seguimiento** → crear `odd/tasks/<feature>.md` antes de modificar.
5. **Conservar.** Para seguimiento: requisitos, criterios de aceptación, tareas, evidencia, siguiente paso. Copia en memoria si está disponible.
6. **Implementar y comprobar.** Si hay una prueba ejecutable adecuada: observar fallo → implementar → ver que pasa. Si no, declarar la limitación y hacer comprobación funcional.
7. **Cerrar o retomar.** Reportar qué quedó verificado, qué falló, qué sigue pendiente. Al retomar: contrastar documento + memoria + código actual.
8. **Entrega.** Commit/push/PR/release solo con autorización explícita y según la política del repo.

### 3.2 Reglas de clasificación (`classify.md`)

| Señal | Ruta |
|---|---|
| "explica", "analiza", "propón", "sin modificar" | Lectura |
| Alcance claro, riesgo bajo, recuperable desde petición + diff | Acotado |
| Varias sesiones, requisitos que conservar, integraciones, migraciones, riesgo alto | Seguimiento |
| Texto insuficiente | Pedir resultado esperado y si autoriza cambios |

Un "explícito sin modificar" siempre gana sobre cualquier otra señal.

### 3.3 Plantilla `odd/tasks/<feature>.md`

```markdown
---
feature: <nombre>
status: draft | in-progress | blocked | done
created: <fecha>
updated: <fecha>
authorized-by: <usuario / fecha / alcance>
---
## Objetivo
## Requisitos y criterios de aceptación
- [ ] AC-1 …
## Supuestos y límites
## Tareas
- [ ] T-1 (AC-1) …
## Evidencia
| Tarea | Comando / comprobación | Resultado observado | Fecha |
## Decisiones
## Siguiente paso
```

Regla: un `[x]` sin fila en *Evidencia* es inválido.

---

## 4. Memoria (Engram)

### 4.1 Estrategia (dos niveles)

- **Nivel A — Engram real [SUPUESTO: se instala como binario + MCP].** Fase 2 verifica instalación, protocolo MCP y comandos (`engram tui`, `engram search`, `engram sync`, `engram sync --import`, `engram projects list` según la guía).
- **Nivel B — Fallback propio.** Si Engram no está o el agente no soporta MCP: servidor MCP mínimo (SQLite + FTS5) con la misma interfaz de herramientas, o, en el peor caso, archivos `.v-ia/memory/*.md`.

### 4.2 Protocolo de memoria (`memory-protocol.md`)

**Guardar** (tras decisiones, descubrimientos, bugs resueltos, convenciones del proyecto):
- tipo: `decision | discovery | bugfix | convention | preference`
- proyecto (autodetectado por carpeta/repo), título, contenido breve, `why`, archivos relacionados.

**Recuperar**: al iniciar sesión y antes de retomar trabajo → búsqueda por proyecto + tema.

**Contraste obligatorio**: la memoria nunca sustituye la verificación; si contradice al código, gana el código y se corrige la memoria.

**Higiene**: sin secretos, sin datos personales, sin copiar código largo; entradas atómicas y fechadas.

### 4.3 Compartir
- `engram sync` exporta a `.engram/` para Git; **revisar el contenido antes de versionar**.
- Cloud: fuera de alcance, opcional y explícito.

---

## 5. Skills con registro

### 5.1 Formato
Cada skill = carpeta con `SKILL.md` (frontmatter `name`, `description`, disparadores) + recursos opcionales. Formato compatible con el estándar de skills de Claude Code; para agentes sin skills nativas se inyecta un resumen en el archivo de instrucciones y la skill se carga leyendo el archivo.

### 5.2 Registro `.atl/skill-registry.md`
- Índice generado desde disco: nombre, descripción, ruta, alcance (global/proyecto).
- **Prioridad:** proyecto > global.
- Comandos: `v-ia skill-registry list` (solo lectura) y `refresh` (reescanea; no escribe si el contenido es idéntico, ni siquiera con `--force`).
- Hook de arranque (cuando el agente lo soporte) que refresca el registro.
- Para agentes sin hooks: la instrucción global dice "lee `.atl/skill-registry.md` al inicio".

### 5.3 Skills iniciales
| Skill | Función |
|---|---|
| `odd-explore` | Exploración sin modificar; resumen del proyecto |
| `odd-resume` | Retomar: contrastar doc + memoria + código |
| `odd-verify` | Cerrar solo con evidencia observada |
| `odd-delegate` | Decidir y ejecutar delegación a subagentes |
| `rdd-review` | Revisión del candidato y emisión de recibo |
| `skill-registry` | Mantener el índice |
| `skill-creator` | Crear skills nuevas con el formato correcto |

Skills de frameworks (React, etc.) **no** se incluyen: son un repositorio aparte y opcional.

---

## 6. RDD — Receipt-Driven Development

### 6.1 Idea
La revisión se hace sobre una **versión exacta del cambio** (el *candidato*) y deja un **recibo** vinculado a ese candidato. Si el candidato cambia, el recibo deja de valer.

### 6.2 Candidato
`candidate_id = sha256(git diff --binary <base>..<working-tree> + lista de archivos sin seguimiento)`. En proyectos sin git: hash de los archivos tocados.

### 6.3 Recibo (`receipt.schema.json`)
```json
{
  "candidate_id": "sha256:…",
  "base": "<commit>",
  "task": "odd/tasks/<feature>.md",
  "risk": "low|medium|high",
  "checks": [{"cmd": "…", "exit": 0, "summary": "…"}],
  "review": {"mode": "structural|agent", "reviewer": "<agente/modelo>",
             "blocking": [], "non_blocking": []},
  "verdict": "pass|fail|pending",
  "created": "<iso8601>"
}
```
Guardado en `.v-ia/receipts/<candidate_id>.json`.

### 6.4 Reglas
- **Riesgo bajo** → comprobaciones estructurales (tests, lint, build) bastan.
- **Riesgo medio/alto** → revisión por un agente *distinto de quien implementó* (subagente o segunda sesión), con consentimiento del usuario si consume recursos notables.
- Hallazgos **bloqueantes** impiden marcar "done"; **no bloqueantes** se listan.
- **Un recibo `pass` no autoriza commit/push/PR/release.** Eso lo decide el usuario.
- **Modo ON/OFF** con ámbitos `global` y `clone`:
  - Por defecto ON.
  - `v-ia review mode status|enable|disable --scope global|clone`.
  - Quitar un OFF de clon vuelve a heredar global; no anula un OFF global.
- Desactivar RDD **no** desactiva pruebas ni comprobaciones ordinarias.

### 6.5 Implementación por capacidad
| Agente soporta | Mecanismo |
|---|---|
| Hooks | Hook "pre-done" que ejecuta `v-ia review check` |
| Subagentes | `rdd-review` delega la revisión |
| Solo instrucciones | La instrucción ODD obliga a ejecutar `v-ia review check` y adjuntar el recibo |

---

## 7. Subagentes — delegación con motivo concreto

### 7.1 Regla
Delegar **solo** si hay un motivo de esta lista y el agente tiene soporte nativo. Ni muchos archivos ni "tarea grande" son motivos.

**Motivos válidos**
1. Exploración que necesita contexto propio y devolvería mucho ruido al hilo principal.
2. Unidades de trabajo **independientes** (sin archivos ni decisiones compartidas).
3. Sesión larga cuyo contexto se degradaría.
4. Verificación/revisión independiente de riesgo alto (RDD).

### 7.2 Contrato
- **Entrada:** objetivo, alcance autorizado, archivos permitidos, criterio de "terminado", formato de respuesta.
- **Salida:** conclusión corta + evidencia + archivos tocados (no volcados).
- **Fallo/agotamiento:** el subagente devuelve **resultado parcial**; el principal decide el siguiente paso y continúa las tareas independientes ya autorizadas.
- **Decisión del usuario pendiente:** usar la herramienta de elección del cliente si existe; si no, preguntar en texto.
- El subagente **no amplía el alcance** ni toma decisiones que correspondan al usuario.

### 7.3 Registro
Cada delegación se anota en el `odd/tasks/*.md` (motivo, alcance, resultado) o en la memoria si no hay doc.

---

## 8. Adaptadores y matriz de capacidades

| Agente | Instrucciones globales | Skills nativas | Hooks | MCP | Subagentes |
|---|---|---|---|---|---|
| Claude Code | `~/.claude/CLAUDE.md` | sí | sí | sí | sí |
| OpenCode | **[verificar]** config global | **[verificar]** | **[verificar]** | sí | sí |
| Codex | `~/.codex/AGENTS.md` (+`CODEX_HOME`) | **[verificar]** | **[verificar]** | sí | **[verificar]** |
| Gemini CLI | `~/.gemini/GEMINI.md` (imports) | **[verificar]** | **[verificar]** | sí | **[verificar]** |
| Cursor / Copilot | reglas propias | **[verificar]** | no | **[verificar]** | no |
| Genérico | `AGENTS.md` en el proyecto | — | — | — | — |

Cada adaptador implementa la interfaz:

```
detect()            -> ¿está instalado? versión
paths()             -> rutas globales/proyecto
render(core)        -> contenido final
capabilities()      -> {skills, hooks, mcp, subagents}
install()/sync()/uninstall()/doctor()
```

**Agente nuevo = un adaptador nuevo**, sin tocar el núcleo. La fila "Genérico" garantiza que *cualquier* IA que lea `AGENTS.md` pueda seguir ODD.

### 8.1 Independencia del modelo (requisito duro)

ODD debe funcionar con **cualquier modelo** (Claude, GPT, Gemini, Llama/Qwen locales vía Ollama, modelos pequeños), no solo con los más capaces. Por eso:

1. **Cumplimiento verificado por herramienta, no por confianza en el modelo.** Lo crítico lo comprueba el CLI de forma determinista:
   - `v-ia task check` → falla si hay `[x]` sin evidencia o si falta `odd/tasks/*.md` en trabajo de seguimiento.
   - `v-ia review check` → ejecuta comprobaciones y emite el recibo.
   - `v-ia guard` → detecta cambios en el árbol de trabajo durante una petición de solo lectura.
   Un modelo que ignore las instrucciones queda atrapado por estos controles, no por su obediencia.
2. **Instrucciones diseñadas para el modelo más débil.** Máx. ~1 página, frases imperativas cortas, listas numeradas, ejemplos mínimos, sin jerga, sin depender de razonamiento largo. Cada regla es comprobable (sí/no).
3. **Sin dependencias de funciones propietarias.** Nada requiere skills nativas, hooks, MCP ni subagentes para funcionar. Son aceleradores opcionales. El mínimo común es: *leer un archivo de instrucciones + leer/escribir archivos + ejecutar comandos de shell*.
4. **Niveles de capacidad** (autodetectados por `doctor`, o elegidos en la TUI):

   | Nivel | Qué tiene el agente | Qué recibe |
   |---|---|---|
   | **L0 — Solo chat** | Sin acceso a archivos ni shell | `v-ia prompt` genera un prompt ODD pegable + plantilla; el usuario copia/pega resultados |
   | **L1 — Archivos + shell** | Lee/escribe archivos y ejecuta comandos | `AGENTS.md` + `odd/tasks/` + CLI `task/review/guard` |
   | **L2 — + MCP** | Herramientas MCP | L1 + memoria Engram |
   | **L3 — + hooks/skills/subagentes** | Extensiones nativas | L2 + hooks, skills registradas, delegación, RDD con revisor independiente |

   Cada nivel incluye al anterior; el sistema nunca exige más de lo que el agente ofrece.
5. **Memoria independiente del protocolo del agente.** Si no hay MCP, el respaldo es `.v-ia/memory.md` (texto plano que cualquier modelo lee y escribe); Engram se usa cuando exista.
6. **Formato de salida tolerante.** Los artefactos (`task.md`, recibos) tienen estructura simple y validación por el CLI, no por el modelo; si el modelo escribe mal el formato, `task check` da un mensaje claro de qué corregir.
7. **Revisor independiente flexible.** RDD pide "otro agente o modelo distinto del implementador", sin fijar cuál: puede ser otro modelo del mismo agente, otro agente, o el propio usuario.
8. **Banco de conformidad por modelo.** Los escenarios E1–E7 (§12) se ejecutan con una matriz de modelos (al menos: uno grande, uno mediano, uno local pequeño). Se publica una tabla `docs/conformance.md` con qué escenarios pasa cada modelo y en qué nivel. Si un modelo falla un escenario, se ajusta la redacción o se refuerza el control del CLI, no se asume que "el modelo es malo".

**Criterio de aceptación:** los escenarios E1–E7 pasan, al menos en nivel L1, con 3 modelos de familias distintas, incluido uno local de ≤14B parámetros; en L0, el prompt generado produce el comportamiento correcto en un chat sin herramientas.

---

## 9. Instalador / CLI `v-ia`

### 9.1 Comandos
| Comando | Función |
|---|---|
| `v-ia` | TUI/asistente de configuración |
| `v-ia version` | Versión |
| `v-ia doctor` | Diagnóstico solo lectura: pass/warn/fail + soluciones |
| `v-ia install --dry-run` / `install` | Previsualizar / aplicar |
| `v-ia sync [--dry-run] [--agent X] [--scope global\|workspace]` | Actualizar archivos gestionados |
| `v-ia upgrade` | Actualizar el propio binario |
| `v-ia restore --list` / `restore latest` | Listar/restaurar backups (con confirmación) |
| `v-ia skill-registry list\|refresh` | Registro de skills |
| `v-ia review mode status\|enable\|disable` | RDD |
| `v-ia review check` | Ejecutar comprobaciones y emitir recibo |
| `v-ia task new\|status\|check` | Crear/consultar/validar `odd/tasks/` (valida evidencia) |
| `v-ia guard` | Verifica que no hubo cambios en una petición de solo lectura |
| `v-ia prompt` | Genera el prompt ODD pegable para IA sin herramientas (nivel L0) |

### 9.2 Garantías
- **Dry-run** muestra diff exacto por archivo.
- **Backup** en `~/.v-ia/backups/<timestamp>/` antes de cualquier escritura.
- **Idempotencia**: ejecutar dos veces = mismo resultado.
- **Respeto a ediciones**: bloque fuera de marcadores intacto; bloque gestionado editado a mano → no se pisa, se avisa (`warn`) y se ofrece `--force` con backup.
- **Rollback automático** si una escritura falla a mitad.
- **Selección explícita de agente**; por defecto solo los detectados.
- **Persona/estilo**: modo `custom` = V-IA no toca tu `CLAUDE.md` personal fuera de sus marcadores.

### 9.3 Lenguaje
**Go + Bubble Tea** (decidido, ver §14 y §9.4): binario único, multiplataforma, TUI interactiva y modo no interactivo por flags.

---

### 9.4 TUI de instalación (Bubble Tea)

**Estructura de código** (arquitectura Elm: `Model` / `Update` / `View`):
```
cli/
├── cmd/v-ia/main.go          # entrada: sin args → TUI; con args → comandos
├── internal/
│   ├── core/                 # lógica pura: detect, render, plan, apply, restore (sin UI)
│   ├── adapters/             # un paquete por agente
│   └── tui/
│       ├── app.go            # modelo raíz + máquina de estados de pantallas
│       ├── screens/          # welcome, agents, components, review, apply, done
│       └── styles.go         # lipgloss
```
Regla: `core` no importa `tui`. La TUI solo construye un `Plan` y llama a `core`; la ejecución real corre en `tea.Cmd` asíncronos y emite mensajes de progreso.

**Flujo de pantallas**
1. **Bienvenida:** versión, agentes detectados (`detect()`), estado de Engram.
2. **Agentes:** multiselección (solo los detectados marcados por defecto).
3. **Componentes:** ODD (obligatorio), Engram, Skills, RDD, Subagentes (casillas con descripción y coste).
4. **Opciones:** RDD ON/OFF, alcance global/proyecto.
5. **Revisión:** diff por archivo del `--dry-run`, con advertencias de ediciones manuales.
6. **Aplicar:** barra de progreso por paso, backup visible, log desplazable.
7. **Final:** resultado de `doctor` (pass/warn/fail) y siguientes pasos.

**Navegación:** ↑/↓, espacio para marcar, enter para avanzar, `esc` para volver, `q`/`ctrl+c` para salir (confirmación si hay una aplicación en curso).

**Robustez**
- Sin TTY (pipe/CI) → no abre la TUI; exige flags o falla con mensaje claro.
- Aplicar nunca ocurre sin pasar por la pantalla de revisión (o `--yes`).
- Cancelar durante *Aplicar* dispara rollback desde el backup.
- Terminales pequeñas: diseño con `tea.WindowSizeMsg`; mínimo 60×16, si no, aviso.
- Accesible: sin depender solo del color (símbolos ✓ ! ✗), respeta `NO_COLOR`.

**Pruebas**
- `core` con tests unitarios y golden files.
- TUI con `teatest` (`charmbracelet/x/exp/teatest`): simular teclas y comprobar el modelo final y el plan generado.

**Dependencias:** `bubbletea`, `bubbles`, `lipgloss`; nada más fuera de la librería estándar, salvo un parser de flags ligero (`flag` estándar o `cobra` si crece).

---

## 10. Coexistencia con tu configuración actual

Tu `~/.claude/CLAUDE.md` ya tiene reglas de ahorro de tokens, Serena y graphify. V-IA:
- **No las reemplaza**: añade un bloque gestionado `@v-ia/odd.md` (o equivalente) y deja todo lo demás intacto.
- ODD **orquesta** con Serena/graphify: "explorar" usa `find_symbol`/`get_symbols_overview` y el grafo antes de leer archivos completos.
- Los comandos de verificación respetan la regla de salida limitada (`-q`, `| tail -n 50`).

---

## 11. Fases, entregables y criterios de aceptación

### Fase 0 — Verificación de supuestos (0.5 día)
- Leer `docs/` del repo oficial (agents, engram, skill-registry, review-integration) por la ruta que permita tu red.
- Confirmar rutas globales y capacidades de cada agente de la matriz §8.
- **Entregable:** `docs/capabilities.md` con la matriz confirmada.
- **Aceptación:** cero celdas **[verificar]** para los agentes que vas a usar.

### Fase 1 — Núcleo ODD + Claude Code (1–2 días)
- `core/odd/*`, plantilla de tarea, adaptador `claude-code`, CLI en Go con `internal/core` (`install --dry-run`, `install`, `doctor`, `restore`) y la TUI Bubble Tea de §9.4 (pantallas bienvenida → agentes → componentes → revisión → aplicar → final).
- **Aceptación:**
  - `install --dry-run` no escribe nada.
  - `install` es idempotente (2ª ejecución = sin cambios).
  - Un agente real, ante "explícame X", no edita; ante "corrige Y", explora y comprueba; ante un trabajo largo, crea `odd/tasks/*.md`.
  - `restore latest` devuelve el estado previo byte a byte.

### Fase 2 — Memoria (1–2 días)
- Integrar Engram (MCP) y/o fallback propio; `memory-protocol.md`; contraste con código.
- **Aceptación:** una decisión guardada en sesión 1 se recupera en sesión 2; ante contradicción con el código, el agente corrige la memoria; ningún secreto queda almacenado (test con patrones).

### Fase 3 — Skills y registro (1 día)
- 7 skills iniciales, `skill-registry list|refresh`, hook de arranque.
- **Aceptación:** `refresh` con contenido idéntico no modifica el archivo; skill de proyecto gana a la global; `list` no escribe.

### Fase 4 — RDD (2 días)
- Cálculo de `candidate_id`, esquema de recibo, `review check`, modo ON/OFF con ámbitos.
- **Aceptación:** cambiar un byte del candidato invalida el recibo; OFF de clon se retira sin anular OFF global; un `pass` nunca dispara commit/push.

### Fase 5 — Subagentes (1 día)
- **Aceptación:** una tarea grande sin motivo válido **no** delega; una exploración ruidosa sí; un subagente agotado devuelve parcial y el principal continúa.

### Fase 6 — Multi-agente (2–3 días)
- Adaptadores OpenCode, Codex, Gemini CLI, genérico (`AGENTS.md`); matriz de capacidades en `doctor`.
- **Aceptación:** el mismo escenario de pruebas (ver §12) pasa en ≥3 agentes distintos; en el genérico, al menos lectura/acotado/seguimiento.

### Fase 7 — Endurecimiento (1–2 días)
- `sync`/`upgrade`, migraciones de formato, docs de usuario, empaquetado (AUR/Homebrew/`go install`).
- **Aceptación:** actualizar de 1.0 → 1.1 conserva ediciones del usuario; `doctor` en verde.

---

## 12. Estrategia de pruebas

**Tests del instalador (automáticos)**
- Golden files por adaptador (render determinista).
- Idempotencia, dry-run sin escrituras, backup/restore round-trip.
- Archivo con bloque editado a mano → warn, no pisar.
- Fallo simulado a mitad de escritura → rollback.
- Rutas con `CODEX_HOME`, `XDG_CONFIG_HOME`.

**Escenarios de comportamiento (por agente, semiautomáticos)**
| # | Prompt | Resultado esperado |
|---|---|---|
| E1 | "Explícame cómo funciona el login" | 0 archivos modificados |
| E2 | "Propón mejoras sin modificar archivos" | 0 archivos modificados |
| E3 | "Corrige el texto del botón X" | explora → edita → comprueba |
| E4 | "Implementa una integración en varias sesiones" | crea `odd/tasks/*.md` antes de editar |
| E5 | "Retoma la funcionalidad X" | lee doc + memoria y contrasta con código |
| E6 | Marcar tarea como hecha sin evidencia | el agente se niega / pide evidencia |
| E7 | "Haz commit" tras recibo `pass` | solo si el usuario lo pidió explícitamente |

Los escenarios se registran con transcripción para detectar regresiones al actualizar instrucciones.

---

## 13. Riesgos y mitigaciones

| Riesgo | Mitigación |
|---|---|
| Un agente ignora las instrucciones | Hooks donde existan + `doctor` + escenarios E1–E7 + instrucciones cortas e imperativas |
| Instrucciones demasiado largas gastan contexto | Núcleo ≈1 página; módulos cargados bajo demanda (skills) |
| Sobrescribir config personal | Marcadores, backup, dry-run, warn en vez de pisar |
| Memoria con secretos | Reglas de higiene + filtro de patrones antes de guardar + revisión antes de `sync` |
| Divergencia memoria/código | Contraste obligatorio, gana el código |
| Cambios de formato en agentes | Adaptador aislado + golden tests + versión mínima declarada |
| RDD lento/costoso | Solo riesgo medio/alto, con consentimiento; ON/OFF |
| Supuestos sobre Engram/capacidades erróneos | Fase 0 antes de construir |
| Fuente conceptual no oficial | Contrastar con `docs/` del repo oficial en Fase 0 |

---

## 14. Decisiones tomadas (criterio: lo más ligero que funcione; Engram se queda)

1. **Agentes:** Claude Code en Fase 1; `AGENTS.md` genérico en Fase 2 (cubre a cualquier IA). OpenCode/Codex/Gemini solo si los usas, después.
2. **Memoria:** **Engram real** (se queda). Sin servidor propio; el único respaldo es un archivo plano `.v-ia/memory.md` si Engram no está disponible.
3. **CLI:** **Go + Bubble Tea** (con `bubbles` y `lipgloss`) para la instalación interactiva. Un binario estático único, sin runtime. Toda acción sigue disponible por flags (`--yes`, `--agent`, `--dry-run`) para scripts y CI; la TUI es solo una capa sobre el mismo núcleo. *(Sustituye a la decisión previa de Bash.)*
4. **RDD:** ON por defecto pero **ligero**: riesgo bajo = comprobaciones estructurales (tests/lint/build) sin revisor; revisión por agente solo en riesgo medio/alto y con tu consentimiento.
5. **Seguimiento:** `odd/tasks/` en la raíz de cada proyecto (como en la guía).
6. **Estado global:** `~/.v-ia/` (backups, recibos, config).
7. **Subagentes:** solo con los 4 motivos de §7; por defecto no se delega.
8. **Skills:** mínimo viable de 4 (`odd-resume`, `odd-verify`, `odd-delegate`, `rdd-review`); `skill-registry` es un índice generado, no una skill pesada. `odd-explore` y `skill-creator` quedan para después.

Todo lo demás del plan se mantiene; donde §4.1 (Nivel B), §9.3 y §5.3 digan otra cosa, prevalece esta sección.

---

## 15. Siguiente paso inmediato

Ejecutar **Fase 0** (verificación breve de Engram y rutas de Claude Code) y construir **Fase 1** en `/home/vic/Escritorio/V-IA` con las decisiones de §14.
