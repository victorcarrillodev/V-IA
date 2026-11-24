// V-IA: hace cumplir ODD en OpenCode por mecanismo (bloquea en vez de pedir).
// Gestionado por `v-ia`; no lo edites (se sobrescribe en `v-ia sync`).
import { spawnSync } from "node:child_process"

const BIN = process.env.V_IA_BIN || __V_IA_EXE__
const EDIT = new Set(["edit", "write", "patch", "multiedit", "apply_patch"])

// Llama al hook de V-IA con el mismo JSON que usa Claude Code. Falla abierto:
// si v-ia no responde, no se bloquea nada.
function run(cwd, payload) {
  const r = spawnSync(BIN, ["hook"], {
    input: JSON.stringify({ cwd, ...payload }),
    encoding: "utf8",
    timeout: 15000,
  })
  if (r.error || r.status === null) return ""
  if (r.status === 2) throw new Error((r.stderr || "V-IA: acción bloqueada").trim())
  return r.stdout || ""
}

export const VIAPlugin = async ({ directory }) => ({
  // Clasifica la petición del usuario (lectura / edición / entrega).
  "chat.message": async (_input, output) => {
    const text = (output.parts || [])
      .filter((p) => p.type === "text" && !p.synthetic)
      .map((p) => p.text)
      .join("\n")
    if (text) run(directory, { hook_event_name: "UserPromptSubmit", prompt: text })
  },
  // Bloquea ediciones en modo lectura, comandos destructivos y entregas no pedidas.
  "tool.execute.before": async (input, output) => {
    const tool = String(input.tool || "").toLowerCase()
    const args = output.args || {}
    if (EDIT.has(tool)) {
      run(directory, {
        hook_event_name: "PreToolUse",
        tool_name: "Edit",
        tool_input: { file_path: args.filePath || args.file_path || args.path || "" },
      })
    } else if (tool === "bash") {
      run(directory, {
        hook_event_name: "PreToolUse",
        tool_name: "Bash",
        tool_input: { command: String(args.command || "") },
      })
    }
  },
})
