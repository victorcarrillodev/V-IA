// Package corefs embebe las instrucciones ODD, skills y subagentes (fuente de verdad) en el binario.
package corefs

import "embed"

//go:embed odd skills agents memory plugins personas
var FS embed.FS
