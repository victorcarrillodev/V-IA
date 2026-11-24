package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"v-ia/internal/adapters"
	"v-ia/internal/extras"
)

// cmdDeps muestra qué falta (Engram, Context7, CodeGraph) y lo instala si lo confirmas.
// Nunca usa sudo: lo que instala va a ~/.local/bin, npm global o la config de Claude Code.
func cmdDeps(args []string) int {
	fs := flag.NewFlagSet("deps", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "instalar lo que falte sin preguntar")
	only := fs.String("only", "", "limitar a: "+strings.Join(extras.All, ", ")+" (por defecto, todos)")
	if fs.Parse(args) != nil {
		return 2
	}
	home, err := adapters.Home()
	if err != nil {
		return fail("%v", err)
	}
	ids := extras.All
	if *only != "" {
		ids = splitList(*only)
	}
	var pending []string
	for _, id := range ids {
		inf, ok := extras.Describe(id)
		if !ok {
			return fail("componente desconocido: %s (disponibles: %s)", id, strings.Join(extras.All, ", "))
		}
		if extras.Installed(home, id) {
			fmt.Printf("✓ %s\n", inf.Title)
		} else {
			fmt.Printf("✗ %s\n", inf.Title)
			pending = append(pending, id)
		}
	}
	if len(pending) == 0 {
		fmt.Println("✓ todo está instalado")
		return 0
	}
	steps, notes, err := extras.Plan(home, pending)
	fmt.Println("\nPara instalar lo que falta se ejecutaría (sin sudo):")
	for _, st := range steps {
		fmt.Printf("  $ %s   # %s\n", st, st.Desc)
	}
	for _, n := range notes {
		fmt.Println("  i", n)
	}
	if err != nil {
		return fail("%v", err)
	}
	if !*yes {
		if fi, e := os.Stdin.Stat(); e != nil || fi.Mode()&os.ModeCharDevice == 0 {
			fmt.Println("\nSin terminal interactiva: repite con --yes para instalar.")
			return 1
		}
		fmt.Print("\n¿Instalar ahora? [s/N] ")
		ans, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(ans)); a != "s" && a != "si" && a != "sí" && a != "y" {
			fmt.Println("cancelado")
			return 1
		}
	}
	if err := extras.Run(steps, os.Stdout); err != nil {
		return fail("%v", err)
	}
	fmt.Println("✓ listo; reinicia tu agente para cargar los servidores MCP")
	return 0
}
