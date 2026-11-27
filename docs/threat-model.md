# Modelo de amenazas de V-IA

Qué impide V-IA, contra quién, y qué **no** impide. Léelo antes de fiarte de un "bloqueado".

## A quién protege y de qué

V-IA trata al **agente de IA como un actor con buena intención pero poco fiable**: se equivoca, interpreta de más o "ayuda" donde no debe. No es una defensa contra un atacante que controla tu máquina ni contra un modelo que busca activamente esquivar los controles.

| Riesgo | Control | Garantía |
|---|---|---|
| Editar cuando solo preguntaste | Hook `PreToolUse` (Claude Code) o plugin `v-ia.js` (OpenCode) | **Fuerte** para las herramientas de edición; **heurística** para Bash (ver abajo) |
| Commit/push/PR no pedidos | Hook sobre `git commit/push/tag`, `gh pr/release` | Fuerte para esos comandos literales |
| Comandos destructivos | Lista de patrones (`rm -rf /`, `push --force`, `reset --hard`…) | Heurística: cubre las formas comunes, no toda variante |
| Cerrar sin evidencia | `task check`, `task verify`, hook `Stop` (solo Claude Code) | Forma + contraste con el recibo; no prueba que el resultado sea correcto |
| Entregar sin revisar | Recibo ligado al hash del árbol + visto bueno del revisor si el riesgo es alto | Un solo byte distinto invalida el recibo |
| Leer o editar secretos | Deny-list en `settings.json` (`~/.ssh`, `.env`, credenciales) | Solo Claude Code |

## Límites que no se pueden cerrar con hooks

- **Bash es un lenguaje entero.** `writesFiles` reconoce redirecciones, `sed -i`, `tee`, `mv/cp/rm`, `git apply`… pero un `python -c "open('f','w')…"` o un script ya existente escribe sin que se vea. En modo lectura el hook `Stop` (Claude Code) lo detecta después con `guard`; en OpenCode no hay `Stop`.
- **La clasificación lectura/edición es por palabras clave** sobre tu mensaje. Una frase ambigua se clasifica mal; el remedio es `v-ia mode …`.
- **Los hooks los ejecuta el cliente.** Si el agente (o tú) edita `~/.claude/settings.json` o borra `plugins/v-ia.js`, el control desaparece. `v-ia doctor` lo detecta, pero no lo impide.
- **Falla abierto.** Si `v-ia` no está en el PATH, se cuelga o devuelve basura, el hook deja pasar la acción: prefiere no bloquear tu trabajo a bloquearlo por un fallo propio.
- **Agentes sin hooks** (Kilo, Qwen, Windsurf, Copilot CLI): solo reciben instrucciones. Cumplirlas depende del modelo.
- **Hooks de Gemini CLI, Codex y Cursor**: el hook de petición de Gemini y Codex se verificó con el agente real; los hooks previos a la herramienta y Cursor, no. Codex avisa de que PreToolUse no intercepta todas las llamadas, y Cursor abre ante fallos de hook salvo `failClosed`.
- **TDD estricto** comprueba que existe una prueba nueva o modificada, no que falle ni que pruebe lo pedido.

## El recibo (RDD)

El recibo liga el resultado de las comprobaciones al **contenido exacto del árbol** (hash de cada archivo no ignorado). Garantiza que "pasó" se refiere a *esta* versión. No garantiza que las comprobaciones sean buenas: si `.v-ia/checks.txt` contiene `true`, el recibo es `pass`. Un recibo `pass` nunca autoriza entregar; eso lo sigue decidiendo tu mensaje.

El revisor independiente (`v-ia-reviewer`) es otra instancia del modelo con solo lectura: reduce el sesgo de autor, no lo elimina.

## Recuento local

`v-ia telemetry on` (apagado por defecto) cuenta cuántas veces bloqueó cada regla y lo guarda en `~/.v-ia/telemetry.json`. **No hay red ni identificador**: solo contadores por tipo de bloqueo, para que veas qué reglas molestan o salvan. `v-ia telemetry reset` los borra.

## Qué se ejecuta en tu máquina

- Los hooks ejecutan `v-ia hook`, sin red.
- `v-ia review check` ejecuta los comandos de `.v-ia/checks.txt` o los inferidos (`go test`, `npm test`): son comandos de tu repositorio, con tus permisos.
- `v-ia deps` ejecuta instaladores oficiales de Engram, Context7 y CodeGraph (con versión fijada); CodeGraph puede descargar y ejecutar un script remoto si no hay `npm`. Nunca usa `sudo`. Se muestra el comando antes y pide confirmación.
- `v-ia update` verifica el `sha256` de la descarga contra `checksums.txt` de la misma release. No hay firma criptográfica de releases todavía.

## Reportar un fallo de seguridad

Abre un issue privado en el repositorio de GitHub (o escribe al autor) con los pasos para reproducirlo. No publiques un exploit funcional en un issue público.
