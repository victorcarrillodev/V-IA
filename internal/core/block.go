// Package core contiene la lógica pura del instalador (sin UI).
package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// Block es un fragmento gestionado por V-IA dentro de un archivo.
type Block struct {
	ID      string
	Version string
	Content string
}

// Status describe qué pasaría al aplicar un bloque sobre un archivo.
type Status int

const (
	Create    Status = iota // el archivo no existe
	Add                     // el archivo existe, sin bloque nuestro
	Update                  // bloque nuestro desactualizado
	Unchanged               // ya está igual
	Conflict                // el bloque gestionado fue editado a mano
)

func (s Status) String() string {
	return [...]string{"create", "add", "update", "unchanged", "conflict"}[s]
}

func sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func normalize(c string) string { return strings.TrimRight(c, "\n") + "\n" }

// Render devuelve el bloque con marcadores.
func Render(b Block) string {
	c := normalize(b.Content)
	return fmt.Sprintf("<!-- v-ia:managed begin id=%s version=%s sha256=%s -->\n%s<!-- v-ia:managed end id=%s -->\n",
		b.ID, b.Version, sum(c), c, b.ID)
}

var reCache sync.Map // id -> *regexp.Regexp

func blockRe(id string) *regexp.Regexp {
	if r, ok := reCache.Load(id); ok {
		return r.(*regexp.Regexp)
	}
	r := compileBlockRe(id)
	reCache.Store(id, r)
	return r
}

func compileBlockRe(id string) *regexp.Regexp {
	return regexp.MustCompile(`(?s)<!-- v-ia:managed begin id=` + regexp.QuoteMeta(id) +
		` version=\S+ sha256=([0-9a-f]+) -->\n(.*?)<!-- v-ia:managed end id=` + regexp.QuoteMeta(id) + ` -->\n`)
}

// Upsert aplica el bloque sobre el texto existente y devuelve el nuevo texto.
// Con force=false un bloque editado a mano no se pisa (Conflict, texto intacto).
func Upsert(existing string, exists bool, b Block, force bool) (string, Status) {
	rendered := Render(b)
	if !exists {
		return rendered, Create
	}
	re := blockRe(b.ID)
	loc := re.FindStringSubmatchIndex(existing)
	if loc == nil {
		sep := ""
		if existing != "" && !strings.HasSuffix(existing, "\n\n") {
			sep = "\n"
			if !strings.HasSuffix(existing, "\n") {
				sep = "\n\n"
			}
		}
		return existing + sep + rendered, Add
	}
	oldSum, oldBody := existing[loc[2]:loc[3]], existing[loc[4]:loc[5]]
	if sum(oldBody) != oldSum && !force {
		return existing, Conflict
	}
	if existing[loc[0]:loc[1]] == rendered {
		return existing, Unchanged
	}
	return existing[:loc[0]] + rendered + existing[loc[1]:], Update
}

// Remove quita el bloque gestionado (lo de fuera no se toca).
func Remove(existing, id string) (string, bool) {
	re := blockRe(id)
	loc := re.FindStringIndex(existing)
	if loc == nil {
		return existing, false
	}
	start := loc[0]
	// Upsert separa con una línea en blanco: se retira junto con el bloque.
	if start >= 2 && existing[start-2:start] == "\n\n" {
		start--
	}
	return existing[:start] + existing[loc[1]:], true
}
