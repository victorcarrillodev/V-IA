package hook

import (
	"os"
	"path/filepath"
	"strings"

	"v-ia/internal/adapters"
	"v-ia/internal/review"
)

// reviewReminder recuerda UNA vez por candidato que hay una revisión independiente debida (riesgo alto)
// y no está aprobada. Nunca ejecuta nada: solo avisa; el estado de la sesión es una pista, no autoridad.
func reviewReminder(root string) string {
	home, err := adapters.Home()
	if err != nil {
		return ""
	}
	if on, _ := review.Mode(root, home); !on || len(review.DetectCommands(root)) == 0 {
		return ""
	}
	a := review.Assess(root, "")
	if !a.ReviewDue || a.Tier == review.TierUnknown { // sin repositorio no hay candidato que revisar
		return ""
	}
	st := review.Status(root)
	if st.State == review.PublicReady || st.State == review.PublicNeedsDecision {
		return ""
	}
	c, err := review.TakeCandidate(root, "")
	if err != nil {
		return ""
	}
	flag := filepath.Join(root, ".v-ia", "reminded")
	if b, err := os.ReadFile(flag); err == nil && strings.TrimSpace(string(b)) == c.ID {
		return ""
	}
	_ = os.MkdirAll(filepath.Dir(flag), 0o755)
	_ = os.WriteFile(flag, []byte(c.ID+"\n"), 0o644)
	return "V-IA/RDD: este cambio es de riesgo " + strings.ToUpper(a.Tier) + " y tiene una revisión independiente debida (" + a.ReviewDueReason + "). Ejecuta `v-ia review check` y `v-ia review status`; lanza v-ia-reviewer por cada lente pendiente. (Aviso único por candidato.)"
}
