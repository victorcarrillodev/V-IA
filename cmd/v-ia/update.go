package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"v-ia/internal/adapters"
	"v-ia/internal/update"
)

// cmdUpdate actualiza el binario desde la última release, con sha256 verificado.
// (upgrade, en cambio, recompila desde el código fuente.)
func cmdUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	check := fs.Bool("check", false, "solo comprobar si hay una versión nueva")
	yes := fs.Bool("yes", false, "actualizar sin preguntar")
	noAtt := fs.Bool("no-attestation", false, "no verificar la atestación de GitHub (p. ej. releases anteriores a que existieran)")
	channel := fs.String("channel", os.Getenv("V_IA_CHANNEL"), "stable | beta (también V_IA_CHANNEL)")
	if fs.Parse(args) != nil {
		return 2
	}
	tag, err := update.LatestChannel(*channel)
	if err != nil {
		return fail("%v", err)
	}
	cur := adapters.Version
	if !update.Newer(cur, tag) {
		fmt.Printf("✓ ya tienes la última versión (v%s; última publicada: %s)\n", cur, tag)
		return 0
	}
	fmt.Printf("Hay una versión nueva: %s (tienes v%s)\n", tag, cur)
	if *check {
		fmt.Println("Ejecuta 'v-ia update' para actualizar.")
		return 0
	}
	if !*yes {
		fmt.Print("¿Actualizar ahora? [s/N] ")
		ans, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(ans)); a != "s" && a != "si" && a != "sí" && a != "y" {
			fmt.Println("cancelado")
			return 1
		}
	}
	self, err := os.Executable()
	if err != nil {
		return fail("%v", err)
	}
	tmp, err := os.MkdirTemp("", "v-ia-update-")
	if err != nil {
		return fail("%v", err)
	}
	defer os.RemoveAll(tmp)
	nb, err := update.Download(tag, runtime.GOOS, runtime.GOARCH, tmp)
	if err != nil {
		return fail("%v", err)
	}
	fmt.Println("✓ descarga verificada (sha256)")
	if update.SignatureConfigured() {
		fmt.Println("✓ firma minisign de checksums.txt verificada")
	} else {
		fmt.Println("! sin clave minisign configurada (V_IA_MINISIGN_PUBKEY o -ldflags): no se verifica la firma de la release")
	}
	switch {
	case *noAtt:
		fmt.Println("! atestación omitida por --no-attestation: la integridad depende solo del hash de la release")
	case update.HaveGH():
		if err := update.VerifyAttestation(update.AssetPath(tmp, runtime.GOOS, runtime.GOARCH)); err != nil {
			return fail("%v\n(si esta release es anterior a las atestaciones, repite con --no-attestation)", err)
		}
		fmt.Println("✓ atestación de GitHub verificada (construido por el workflow de release)")
	default:
		fmt.Println("! sin 'gh' no se verifica la atestación; instala la CLI de GitHub para una verificación más fuerte")
	}
	if err := update.Replace(self, nb); err != nil {
		return fail("%v", err)
	}
	fmt.Printf("✓ actualizado a %s en %s\nEjecuta 'v-ia sync --yes' para refrescar los archivos gestionados.\n", tag, self)
	return 0
}
