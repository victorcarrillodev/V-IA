package adapters

import (
	"strings"

	corefs "v-ia/core"
	"v-ia/internal/core"
	"v-ia/internal/extras"
)

// Block son las reglas de este agente: solo prometen bloqueo donde se comprobó con el
// agente real, solo nombran subagentes donde V-IA los instala y no repiten el protocolo
// de memoria si Engram está configurado o se va a configurar (planned).
func (a Adapter) Block(home string, planned bool) (core.Block, error) {
	b, err := ODDBlock(a.HooksVerified)
	if err != nil {
		return b, err
	}
	// Engram omite Codex sin su ejecutable, así que ahí lo planeado no cuenta.
	if (planned && (a.Name != "codex" || codexAvailable())) || extras.HasServer(home, a.Name, "engram") {
		full, err := corefs.FS.ReadFile("memory/memory-protocol.md")
		if err != nil {
			return core.Block{}, err
		}
		short, err := corefs.FS.ReadFile("memory/memory-engram.md")
		if err != nil {
			return core.Block{}, err
		}
		b.Content = strings.Replace(b.Content, string(full), string(short), 1)
	}
	if a.AgentsDir != nil {
		s, err := corefs.FS.ReadFile("odd/subagents.md")
		if err != nil {
			return core.Block{}, err
		}
		b.Content += string(s)
	}
	return b, nil
}

func codexAvailable() bool { _, err := extras.LookPath("codex"); return err == nil }
