// Package corefs embebe las instrucciones ODD (fuente de verdad) en el binario.
package corefs

import "embed"

//go:embed odd
var FS embed.FS
