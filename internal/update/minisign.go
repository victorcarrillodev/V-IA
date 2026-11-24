package update

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	minisign "github.com/jedisct1/go-minisign"
)

// MinisignPublicKeys es el ancla de confianza de las releases: una o dos claves públicas
// minisign (la línea base64 de `minisign -P`) separadas por coma. Se inyecta al compilar:
//
//	-ldflags "-X v-ia/internal/update.MinisignPublicKeys=${CLAVES}"
//
// o, sin recompilar, con V_IA_MINISIGN_PUBKEY. Vacía = sin verificación de firma (se avisa).
// Con dos claves se permite rotar: basta que firme una.
var MinisignPublicKeys = ""

var keyRe = regexp.MustCompile(`^[A-Za-z0-9+/]{56}$`)

func configuredKeys() ([]minisign.PublicKey, error) {
	raw := os.Getenv("V_IA_MINISIGN_PUBKEY")
	if raw == "" {
		raw = MinisignPublicKeys
	}
	if raw == "" {
		return nil, nil
	}
	var keys []minisign.PublicKey
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if !keyRe.MatchString(p) {
			return nil, errors.New("clave pública minisign mal formada (se esperan 56 caracteres base64)")
		}
		k, err := minisign.NewPublicKey(p)
		if err != nil {
			return nil, fmt.Errorf("clave pública minisign: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// SignatureConfigured indica si hay clave configurada, es decir, si Download exige firma.
func SignatureConfigured() bool { k, err := configuredKeys(); return err != nil || len(k) > 0 }

// verifySums comprueba checksums.txt contra checksums.txt.minisig con alguna clave configurada.
// Sin claves devuelve nil (no exigible); con claves, la firma es obligatoria.
func verifySums(sums []byte, base string) error {
	keys, err := configuredKeys()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	rc, err := get(base + "/checksums.txt.minisig")
	if err != nil {
		return fmt.Errorf("falta la firma checksums.txt.minisig: %w", err)
	}
	blob, _ := io.ReadAll(io.LimitReader(rc, 16<<10))
	rc.Close()
	sig, err := minisign.DecodeSignature(string(blob))
	if err != nil {
		return fmt.Errorf("firma minisign ilegible: %w", err)
	}
	for _, k := range keys {
		if ok, _ := k.Verify(sums, sig); ok {
			return nil
		}
	}
	return errors.New("la firma minisign de checksums.txt no es válida con ninguna clave configurada: descarga rechazada")
}
