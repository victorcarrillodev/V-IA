// Package telemetry cuenta, solo en local y solo si lo activas, qué bloquea V-IA y qué
// comandos usas. No hay red: los datos viven en ~/.v-ia/telemetry.json y los ves con
// `v-ia telemetry show`. Sirve para afinar las reglas, no para enviar nada a nadie.
package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type Data struct {
	Counts map[string]int `json:"counts"`
}

var mu sync.Mutex

func flag(home string) string { return filepath.Join(home, ".v-ia", "telemetry") }
func file(home string) string { return filepath.Join(home, ".v-ia", "telemetry.json") }

// Enabled indica si el usuario activó el recuento local (apagado por defecto).
func Enabled(home string) bool {
	b, err := os.ReadFile(flag(home))
	return err == nil && string(b) == "on\n"
}

// Set activa o desactiva el recuento.
func Set(home string, on bool) error {
	if err := os.MkdirAll(filepath.Dir(flag(home)), 0o755); err != nil {
		return err
	}
	v := "off\n"
	if on {
		v = "on\n"
	}
	return os.WriteFile(flag(home), []byte(v), 0o644)
}

// Load lee los contadores acumulados.
func Load(home string) Data {
	d := Data{Counts: map[string]int{}}
	if b, err := os.ReadFile(file(home)); err == nil {
		_ = json.Unmarshal(b, &d)
	}
	if d.Counts == nil {
		d.Counts = map[string]int{}
	}
	return d
}

// Record suma 1 a un evento. No hace nada si el recuento está apagado; nunca falla hacia fuera.
func Record(home, event string) {
	if !Enabled(home) {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	d := Load(home)
	d.Counts[event]++
	b, _ := json.MarshalIndent(d, "", "  ")
	tmp := file(home) + ".tmp"
	if os.WriteFile(tmp, append(b, '\n'), 0o644) == nil {
		_ = os.Rename(tmp, file(home)) // atómico: otro proceso nunca lee un JSON a medias
	}
}

// Reset borra los contadores.
func Reset(home string) error {
	err := os.Remove(file(home))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Sorted devuelve los eventos ordenados por nombre.
func (d Data) Sorted() []string {
	var k []string
	for e := range d.Counts {
		k = append(k, e)
	}
	sort.Strings(k)
	return k
}
