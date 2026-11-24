package lifecycle

import (
	"fmt"
	"path/filepath"
	"strings"

	corefs "v-ia/core"
	"v-ia/internal/adapters"
)

// agentDef describe un subagente de V-IA. El cuerpo es común; el frontmatter lo genera cada agente.
type agentDef struct {
	Name, File, Desc string
	ReadOnly         bool   // sin edición de archivos
	Bash             bool   // necesita ejecutar `v-ia` (revisor) o comandos (corrector)
	Tools            string // herramientas de Claude Code
}

var agentDefs = []agentDef{
	{"v-ia-reviewer", "agents/v-ia-reviewer.md", "Revisor independiente RDD de una lente (4R), solo lectura sobre el candidato congelado. Registra su revisión con v-ia review ack.", true, true, "Read, Grep, Glob, Bash"},
	{"v-ia-judge-a", "agents/v-ia-judge.md", "Juez ciego A de Judgment Day: revisión adversarial de solo lectura, sin ver al otro juez.", true, false, "Read, Grep, Glob"},
	{"v-ia-judge-b", "agents/v-ia-judge.md", "Juez ciego B de Judgment Day: revisión adversarial de solo lectura, sin ver al otro juez.", true, false, "Read, Grep, Glob"},
	{"v-ia-fixer", "agents/v-ia-fixer.md", "Corrección acotada: arregla solo los hallazgos confirmados dentro de las superficies de edición permitidas.", false, true, "Read, Edit, Write, Glob, Grep, Bash"},
	{"v-ia-explorer", "agents/v-ia-explorer.md", "Explorador de solo lectura: devuelve un mapa corto con ruta:línea para que el padre decida o enrute.", true, false, "Read, Grep, Glob"},
}

// renderAgent devuelve el archivo de un subagente con el frontmatter del estilo del agente anfitrión.
func renderAgent(a agentDef, style string) (string, error) {
	body, err := corefs.FS.ReadFile(a.File)
	if err != nil {
		return "", err
	}
	var fm strings.Builder
	fm.WriteString("---\n")
	switch style {
	case "claude":
		fmt.Fprintf(&fm, "name: %s\ndescription: %s\ntools: %s\n", a.Name, a.Desc, a.Tools)
	case "opencode":
		fmt.Fprintf(&fm, "description: %s\nmode: subagent\npermission:\n", a.Desc)
		if a.ReadOnly {
			fm.WriteString("  edit: deny\n")
		}
		switch {
		case a.Bash && a.ReadOnly:
			fm.WriteString("  bash:\n    \"v-ia *\": allow\n    \"git *\": allow\n    \"*\": deny\n")
		case !a.Bash:
			fm.WriteString("  bash: deny\n")
		}
		fm.WriteString("  task: deny\n")
	case "cursor":
		fmt.Fprintf(&fm, "name: %s\ndescription: %s\nmodel: inherit\nreadonly: %t\nbackground: false\n", a.Name, a.Desc, a.ReadOnly && !a.Bash)
	default:
		return "", fmt.Errorf("estilo de agente desconocido: %s", style)
	}
	fm.WriteString("---\n")
	return fm.String() + string(body), nil
}

// agentFiles son los subagentes de un agente anfitrión (vacío si no admite subagentes en archivos).
func agentFiles(o Options, a adapters.Adapter) ([]managedFile, error) {
	if a.AgentsDir == nil || o.NoSkills {
		return nil, nil
	}
	var out []managedFile
	for _, d := range agentDefs {
		c, err := renderAgent(d, a.AgentStyle)
		if err != nil {
			return nil, err
		}
		out = append(out, managedFile{path: filepath.Join(a.AgentsDir(o.Home), d.Name+".md"), content: c})
	}
	return out, nil
}
