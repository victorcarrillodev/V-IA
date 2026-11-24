---
name: go-testing
description: Escribir y ejecutar pruebas en Go (tabla, subtests, httptest, golden, Bubbletea, race). Úsala al añadir o cambiar código Go o al corregir un fallo de prueba.
license: "Apache-2.0"
metadata:
  author: "v-ia"
  version: "1.0"
---
# Pruebas en Go

1. **Primero la prueba** si el comportamiento está claro: mírala fallar por la razón correcta (`go test -run '^TestX$' ./pkg`), y solo entonces implementa.
2. **Tabla + subtests**: un caso por regla pedida y uno por comportamiento previo que sigue valiendo. Nada de relleno.
3. **Aislar el entorno**: `t.TempDir()`, `t.Setenv("HOME", dir)`, nunca el HOME real. Sustituye `exec`/red por variables de función o `httptest.NewServer`. Las pruebas de integración se saltan con `testing.Short()`.
4. **Errores** con `errors.Is/As`, no por texto. `t.Fatalf` por precondición, `t.Errorf` por comprobación.
5. **Concurrencia**: `go test -race ./...` si hay goroutines, mapas compartidos o caché.
6. **Bubbletea**: prueba `Model.Update()` directamente; usa `teatest` solo para flujos interactivos.
7. **Golden files** solo para salidas largas y estables: se regeneran con `-update`, se inspecciona el diff y se vuelve a ejecutar SIN `-update`.
8. **Cierra con evidencia**: `go build ./... && go vet ./... && go test -count=1 ./...`. Informa los tests cambiados, los escenarios, los comandos, los golden actualizados y la integración omitida.
