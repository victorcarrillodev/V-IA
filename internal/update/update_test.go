package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewer(t *testing.T) {
	casos := []struct {
		cur, lat string
		want     bool
	}{
		{"0.3.0", "v0.4.0", true}, {"0.3.0", "v0.3.0", false}, {"0.3.1", "v0.3.0", false},
		{"0.9.0", "v0.10.0", true}, {"1.0.0", "v0.99.9", false}, {"0.3.0", "v1.0.0-rc1", true},
		{"dev", "v1.0.0", false}, {"0.3.0", "basura", false},
	}
	for _, c := range casos {
		if got := Newer(c.cur, c.lat); got != c.want {
			t.Errorf("Newer(%q,%q)=%v, quería %v", c.cur, c.lat, got, c.want)
		}
	}
}

func tarball(t *testing.T, name, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg})
	tw.Write([]byte(content))
	tw.Close()
	zw.Close()
	return buf.Bytes()
}

func server(t *testing.T, tag string, tgz []byte, sumOverride string) {
	t.Helper()
	asset := "v-ia_linux_amd64.tar.gz"
	sum := sha256.Sum256(tgz)
	hash := hex.EncodeToString(sum[:])
	if sumOverride != "" {
		hash = sumOverride
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/"+Repo+"/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		if tag == "" {
			http.NotFound(w, nil)
			return
		}
		fmt.Fprintf(w, `{"tag_name":%q}`, tag)
	})
	mux.HandleFunc("/"+Repo+"/releases/download/"+tag+"/checksums.txt", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, "%s  %s\n%s  otro.tar.gz\n", hash, asset, strings.Repeat("0", 64))
	})
	mux.HandleFunc("/"+Repo+"/releases/download/"+tag+"/"+asset, func(w http.ResponseWriter, _ *http.Request) { w.Write(tgz) })
	srv := httptest.NewServer(mux)
	oa, od := APIBase, DownloadBase
	APIBase, DownloadBase = srv.URL, srv.URL
	t.Cleanup(func() { srv.Close(); APIBase, DownloadBase = oa, od })
}

func TestLatestSinReleases(t *testing.T) {
	server(t, "", nil, "")
	if _, err := Latest(); err == nil || !strings.Contains(err.Error(), "ninguna release") {
		t.Fatalf("err=%v", err)
	}
}

func TestActualizaYReemplazaAtomicamente(t *testing.T) {
	server(t, "v0.4.0", tarball(t, "v-ia", "BINARIO NUEVO"), "")
	if tag, err := Latest(); err != nil || tag != "v0.4.0" {
		t.Fatalf("tag=%q err=%v", tag, err)
	}
	dir := t.TempDir()
	nb, err := Download("v0.4.0", "linux", "amd64", dir)
	if err != nil {
		t.Fatal(err)
	}
	self := filepath.Join(t.TempDir(), "v-ia")
	os.WriteFile(self, []byte("VIEJO"), 0o755)
	if err := Replace(self, nb); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(self)
	st, _ := os.Stat(self)
	if string(b) != "BINARIO NUEVO" || st.Mode()&0o111 == 0 {
		t.Fatalf("contenido=%q modo=%v", b, st.Mode())
	}
	if _, err := os.Stat(self + ".new"); !os.IsNotExist(err) {
		t.Fatal("no debe quedar el temporal")
	}
}

func TestHashManipuladoSeRechazaYNoTocaNada(t *testing.T) {
	server(t, "v0.4.0", tarball(t, "v-ia", "MALICIOSO"), strings.Repeat("a", 64))
	dir := t.TempDir()
	if _, err := Download("v0.4.0", "linux", "amd64", dir); err == nil || !strings.Contains(err.Error(), "hash no coincide") {
		t.Fatalf("debía rechazar: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "v-ia.new")); !os.IsNotExist(err) {
		t.Fatal("no debe extraerse nada si el hash falla")
	}
}

func TestTarSinBinarioOConRutaPeligrosa(t *testing.T) {
	server(t, "v0.4.0", tarball(t, "../../etc/otro", "x"), "")
	if _, err := Download("v0.4.0", "linux", "amd64", t.TempDir()); err == nil || !strings.Contains(err.Error(), "no contiene") {
		t.Fatalf("err=%v", err)
	}
}

func TestAssetSinHashEnChecksums(t *testing.T) {
	server(t, "v0.4.0", tarball(t, "v-ia", "x"), "")
	if _, err := Download("v0.4.0", "darwin", "arm64", t.TempDir()); err == nil {
		t.Fatal("sin hash para ese asset debía fallar")
	}
}

func TestEtiquetaInvalidaSeRechaza(t *testing.T) {
	for _, tag := range []string{"../../x", "v1.2", "v1.2.3/../../a", "latest", "v1.2.3 ", "v1.2.3?x=1"} {
		if _, err := Download(tag, "linux", "amd64", t.TempDir()); err == nil || !strings.Contains(err.Error(), "inválida") {
			t.Errorf("tag %q debía rechazarse: %v", tag, err)
		}
	}
	// y la API no puede colarla
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/"+Repo+"/releases/latest", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `{"tag_name":"../../evil"}`) })
	srv := httptest.NewServer(mux)
	defer srv.Close()
	old := APIBase
	APIBase = srv.URL
	defer func() { APIBase = old }()
	if _, err := Latest(); err == nil || !strings.Contains(err.Error(), "inesperada") {
		t.Fatalf("Latest debía rechazar la etiqueta: %v", err)
	}
}

func TestChecksumsContradictoriosSeRechazan(t *testing.T) {
	tgz := tarball(t, "v-ia", "x")
	mux := http.NewServeMux()
	asset := "v-ia_linux_amd64.tar.gz"
	mux.HandleFunc("/"+Repo+"/releases/download/v0.4.0/checksums.txt", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, "%s  %s\n%s  %s\n", strings.Repeat("a", 64), asset, strings.Repeat("b", 64), asset)
	})
	mux.HandleFunc("/"+Repo+"/releases/download/v0.4.0/"+asset, func(w http.ResponseWriter, _ *http.Request) { w.Write(tgz) })
	srv := httptest.NewServer(mux)
	defer srv.Close()
	old := DownloadBase
	DownloadBase = srv.URL
	defer func() { DownloadBase = old }()
	if _, err := Download("v0.4.0", "linux", "amd64", t.TempDir()); err == nil || !strings.Contains(err.Error(), "contradictorios") {
		t.Fatalf("err=%v", err)
	}
}

func TestBinarioDemasiadoGrandeSeRechaza(t *testing.T) {
	old := maxBin
	maxBin = 1024
	defer func() { maxBin = old }()
	server(t, "v0.4.0", tarball(t, "v-ia", strings.Repeat("A", 4096)), "")
	dir := t.TempDir()
	if _, err := Download("v0.4.0", "linux", "amd64", dir); err == nil {
		t.Fatal("un binario por encima del tope debía rechazarse")
	}
	if _, err := os.Stat(filepath.Join(dir, "v-ia.new")); !os.IsNotExist(err) {
		t.Fatal("no debe quedar un binario truncado")
	}
}

func TestEnlaceSimbolicoDentroDelTarSeIgnora(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	tw.WriteHeader(&tar.Header{Name: "v-ia", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"})
	tw.Close()
	zw.Close()
	server(t, "v0.4.0", buf.Bytes(), "")
	if _, err := Download("v0.4.0", "linux", "amd64", t.TempDir()); err == nil || !strings.Contains(err.Error(), "no contiene") {
		t.Fatalf("un symlink no es el binario: %v", err)
	}
}

func TestVerifyAttestationUsaGhConElRepoCorrecto(t *testing.T) {
	old := RunGH
	defer func() { RunGH = old }()
	var got []string
	RunGH = func(args ...string) ([]byte, error) { got = args; return []byte("ok"), nil }
	if err := VerifyAttestation("/x/v-ia_linux_amd64.tar.gz"); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "attestation verify /x/v-ia_linux_amd64.tar.gz --repo "+Repo {
		t.Fatalf("args=%v", got)
	}
	RunGH = func(args ...string) ([]byte, error) { return []byte("no attestations found"), fmt.Errorf("exit 1") }
	if err := VerifyAttestation("f"); err == nil || !strings.Contains(err.Error(), "no attestations") {
		t.Fatalf("debía propagar el fallo: %v", err)
	}
}
