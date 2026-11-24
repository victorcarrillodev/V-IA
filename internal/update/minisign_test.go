package update

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// signLegacy produce un par (clave pública, firma) en el formato minisign legado ("Ed").
func signLegacy(t *testing.T, msg []byte) (pub, sigFile string) {
	t.Helper()
	pk, sk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyID := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	pub = base64.StdEncoding.EncodeToString(append(append([]byte("Ed"), keyID...), pk...))
	sig := ed25519.Sign(sk, msg)
	trusted := "release de prueba"
	global := ed25519.Sign(sk, append(append([]byte{}, sig...), trusted...))
	sigFile = "untrusted comment: prueba\n" +
		base64.StdEncoding.EncodeToString(append(append([]byte("Ed"), keyID...), sig...)) + "\n" +
		"trusted comment: " + trusted + "\n" +
		base64.StdEncoding.EncodeToString(global) + "\n"
	return
}

func TestVerifySums(t *testing.T) {
	sums := []byte("abc  v-ia_linux_amd64.tar.gz\n")
	pub, sig := signLegacy(t, sums)
	_, otraSig := signLegacy(t, sums) // firmada con otra clave
	serve := func(body string) string {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if body == "" {
				http.NotFound(w, r)
				return
			}
			w.Write([]byte(body))
		}))
		t.Cleanup(srv.Close)
		return srv.URL
	}
	t.Setenv("V_IA_MINISIGN_PUBKEY", "")
	if err := verifySums(sums, serve("")); err != nil || SignatureConfigured() {
		t.Fatalf("sin clave no debe exigir firma: %v", err)
	}
	t.Setenv("V_IA_MINISIGN_PUBKEY", pub)
	if err := verifySums(sums, serve(sig)); err != nil {
		t.Fatalf("firma válida rechazada: %v", err)
	}
	if err := verifySums([]byte("manipulado"), serve(sig)); err == nil {
		t.Fatal("checksums manipulado debía rechazarse")
	}
	if err := verifySums(sums, serve(otraSig)); err == nil {
		t.Fatal("firma de otra clave debía rechazarse")
	}
	if err := verifySums(sums, serve("")); err == nil || !strings.Contains(err.Error(), "falta la firma") {
		t.Fatalf("firma ausente con clave configurada debía fallar: %v", err)
	}
	t.Setenv("V_IA_MINISIGN_PUBKEY", "corta")
	if err := verifySums(sums, serve(sig)); err == nil {
		t.Fatal("clave mal formada debía fallar")
	}
}
