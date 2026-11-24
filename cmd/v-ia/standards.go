package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"v-ia/internal/lifecycle"
	"v-ia/internal/standards"
)

const standardsUsage = "uso: standards init [--force] | check [--head] [--strict] [--no-ai] [--hook] | install [--strict] | off | status"

func cmdStandards(args []string) int {
	root, _ := os.Getwd()
	if len(args) == 0 {
		return fail(standardsUsage)
	}
	switch args[0] {
	case "init":
		fs := flag.NewFlagSet("init", flag.ContinueOnError)
		force := fs.Bool("force", false, "sobrescribir las reglas existentes")
		if fs.Parse(args[1:]) != nil {
			return 2
		}
		p := filepath.Join(root, standards.RulesFile)
		if _, err := os.Stat(p); err == nil && !*force {
			return fail("%s ya existe (usa --force para sobrescribir)", standards.RulesFile)
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return fail("%v", err)
		}
		if err := os.WriteFile(p, []byte(standards.Template), 0o644); err != nil {
			return fail("%v", err)
		}
		fmt.Println("✓ creado", standards.RulesFile, "(edítalo: borra los ejemplos y escribe tus reglas)")
		fmt.Println("  siguiente: v-ia standards check   ·   v-ia standards install   (pre-commit)")
		return 0
	case "check":
		return standardsCheck(root, args[1:])
	case "install":
		fs := flag.NewFlagSet("install", flag.ContinueOnError)
		strict := fs.Bool("strict", false, "bloquear la entrega si hay incumplimientos")
		if fs.Parse(args[1:]) != nil {
			return 2
		}
		if _, found, err := standards.Load(root); err != nil {
			return fail("%v", err)
		} else if !found {
			return fail("no hay %s: ejecuta primero 'v-ia standards init'", standards.RulesFile)
		}
		p, err := standards.InstallHook(root, lifecycle.ExeForHooks(), *strict)
		if err != nil {
			return fail("%v", err)
		}
		mode := "solo avisa"
		if *strict {
			mode = "BLOQUEA si hay incumplimientos"
		}
		fmt.Printf("✓ pre-commit instalado en %s (%s)\n  para saltarlo: variable V_IA_SKIP_STANDARDS=1 u opción --no-verify de git\n", p, mode)
		return 0
	case "off":
		ok, err := standards.RemoveHook(root)
		if err != nil {
			return fail("%v", err)
		}
		if ok {
			fmt.Println("✓ pre-commit de standards retirado (tus reglas se conservan)")
		} else {
			fmt.Println("= no había hook de standards")
		}
		return 0
	case "status":
		rules, found, err := standards.Load(root)
		if err != nil {
			return fail("%v", err)
		}
		if !found {
			fmt.Println("! sin reglas: v-ia standards init")
		} else {
			n := map[standards.Kind]int{}
			for _, r := range rules {
				n[r.Kind]++
			}
			fmt.Printf("✓ %d reglas: %d path, %d deny (sin tokens), %d ai (con agente)\n", len(rules), n[standards.Path], n[standards.Deny], n[standards.AI])
		}
		if on, strict := standards.HookState(root); on {
			fmt.Printf("✓ pre-commit instalado (strict=%v)\n", strict)
		} else {
			fmt.Println("= pre-commit no instalado: v-ia standards install")
		}
		fmt.Printf("= caché: %d revisiones guardadas\n", standards.CacheSize(root))
		return 0
	}
	return fail(standardsUsage)
}

func standardsCheck(root string, args []string) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	head := fs.Bool("head", false, "revisar el árbol de trabajo (diff contra HEAD) en vez de lo que está en staging")
	strict := fs.Bool("strict", false, "salir con error si hay incumplimientos")
	noAI := fs.Bool("no-ai", false, "solo reglas locales (sin tokens)")
	hook := fs.Bool("hook", false, "modo pre-commit: silencioso si no hay nada que decir y nunca falla por infraestructura")
	timeout := fs.Duration("timeout", 90*time.Second, "tiempo máximo de la llamada al agente")
	model := fs.String("model", os.Getenv("V_IA_STANDARDS_MODEL"), "modelo del agente (por defecto haiku)")
	if fs.Parse(args) != nil {
		return 2
	}
	rules, found, err := standards.Load(root)
	switch {
	case err != nil && *hook:
		fmt.Fprintln(os.Stderr, "! standards: reglas ilegibles, se omite:", err)
		return 0
	case err != nil:
		return fail("%v", err)
	case !found && *hook:
		return 0
	case !found:
		return fail("no hay %s: ejecuta 'v-ia standards init'", standards.RulesFile)
	}
	src := standards.Staged
	if *head {
		src = standards.Head
	}
	files, err := standards.Changes(root, src)
	if err != nil {
		if *hook {
			fmt.Fprintln(os.Stderr, "! standards: no se pudo leer el diff, se omite:", err)
			return 0
		}
		return fail("git diff: %v", err)
	}
	if len(files) == 0 {
		if !*hook {
			fmt.Println("= nada que revisar (sin cambios en", string(src)+")")
		}
		return 0
	}
	findings := standards.Deterministic(rules, files)
	var st standards.Stats
	if !*noAI && standards.HasAI(rules) {
		ai, s, err := standards.ReviewAI(root, rules, files, standards.ClaudeRunner(*model), *timeout)
		st = s
		findings = append(findings, ai...)
		if err != nil { // fail-open: una caída del agente nunca bloquea
			fmt.Fprintln(os.Stderr, "! standards: revisión con IA omitida:", err)
		}
	}
	for _, f := range findings {
		fmt.Println("✗", f)
	}
	switch {
	case len(findings) == 0 && !*hook:
		fmt.Printf("✓ cumple los estándares (%d archivos", len(files))
		if st.Pairs > 0 {
			fmt.Printf("; IA: %d revisiones, %d de caché", st.Pairs, st.Cached)
		}
		fmt.Println(")")
	case len(findings) > 0:
		fmt.Printf("! %d incumplimiento(s) de estándares\n", len(findings))
	}
	if st.Skipped > 0 {
		fmt.Printf("! %d archivo(s) no se revisaron con IA por tamaño\n", st.Skipped)
	}
	if *strict && len(findings) > 0 {
		return 1
	}
	return 0
}
