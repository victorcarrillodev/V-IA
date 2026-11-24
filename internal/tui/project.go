package tui

import (
	"fmt"
	"os"
	"strings"

	"v-ia/internal/hook"
	"v-ia/internal/review"
	"v-ia/internal/task"
)

// projectInfo es la foto de solo lectura del proyecto (directorio actual).
type projectInfo struct {
	Dir      string
	Mode     string
	RDD      string
	Tasks    []string
	Receipt  string
	Risk     string
	HasState bool
}

func loadProject(dir, home string) projectInfo {
	p := projectInfo{Dir: dir}
	st := hook.LoadState(dir)
	p.Mode = st.Mode
	if p.Mode == "" {
		p.Mode = "sin clasificar"
	}
	on, src := review.Mode(dir, home)
	p.RDD = fmt.Sprintf("%s (%s)", map[bool]string{true: "ON", false: "OFF"}[on], src)
	p.Tasks, _ = task.Status(dir)
	if r, err := review.Load(dir); err != nil {
		p.Receipt = "sin recibo (ejecuta: v-ia review check)"
	} else {
		state := "vigente"
		if review.Verify(dir, false) != nil {
			state = "obsoleto o fallido"
		}
		ack := ""
		if len(r.SelectedLenses) > 0 {
			ack = fmt.Sprintf(" · revisión: %s (%d/%d lentes)", review.Status(dir).State, len(r.Reviews), len(r.SelectedLenses))
		}
		p.Receipt = fmt.Sprintf("%s · %s · candidato %.10s%s", r.Result, state, r.CandidateID, ack)
	}
	a := review.Assess(dir, "")
	p.Risk = a.Tier
	if len(a.Reasons) > 0 {
		p.Risk += " — " + strings.Join(a.Reasons, "; ")
	}
	if _, err := os.Stat(dir); err == nil {
		p.HasState = true
	}
	return p
}
