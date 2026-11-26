// Package update actualiza el binario v-ia desde la última release de GitHub,
// verificando su sha256 contra checksums.txt antes de reemplazar nada.
package update

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const Repo = "victorcarrillodev/V-IA"

// Sustituibles en pruebas.
var (
	APIBase      = "https://api.github.com"
	DownloadBase = "https://github.com"
	Client       = &http.Client{Timeout: 60 * time.Second}
)

var maxBin int64 = 200 << 20 // tope de tamaño del binario extraído (var: se reduce en pruebas)

var tagRe = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$`)

// Latest devuelve la etiqueta de la última release (por ejemplo "v0.4.0").
func Latest() (string, error) {
	resp, err := Client.Get(fmt.Sprintf("%s/repos/%s/releases/latest", APIBase, Repo))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", errors.New("no hay ninguna release publicada (o el repositorio es privado)")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub respondió %s", resp.Status)
	}
	var r struct {
		Tag string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&r); err != nil || r.Tag == "" {
		return "", errors.New("respuesta de GitHub sin tag_name")
	}
	if !tagRe.MatchString(r.Tag) {
		return "", fmt.Errorf("etiqueta de versión inesperada %q", r.Tag)
	}
	return r.Tag, nil
}

// LatestChannel devuelve la última etiqueta del canal: "stable" (por defecto) usa la
// última release estable; "beta" (alias "nightly") incluye prereleases y elige la de
// versión más alta, sin contar borradores.
func LatestChannel(channel string) (string, error) {
	switch channel {
	case "", "stable":
		return Latest()
	case "beta", "nightly":
	default:
		return "", fmt.Errorf("canal desconocido %q (usa stable o beta)", channel)
	}
	resp, err := Client.Get(fmt.Sprintf("%s/repos/%s/releases?per_page=30", APIBase, Repo))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub respondió %s", resp.Status)
	}
	var rs []struct {
		Tag   string `json:"tag_name"`
		Draft bool   `json:"draft"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&rs); err != nil {
		return "", errors.New("respuesta de GitHub ilegible")
	}
	best := ""
	for _, r := range rs {
		if r.Draft || !tagRe.MatchString(r.Tag) {
			continue
		}
		if best == "" || Newer(best, r.Tag) {
			best = r.Tag
		}
	}
	if best == "" {
		return "", errors.New("no hay ninguna release publicada (o el repositorio es privado)")
	}
	return best, nil
}

func parse(v string) (n [3]int, ok bool) {
	parts := strings.SplitN(strings.TrimPrefix(strings.TrimSpace(v), "v"), ".", 3)
	if len(parts) != 3 {
		return n, false
	}
	for i, p := range parts {
		p, _, _ = strings.Cut(p, "-") // 1.2.3-rc1 → 3
		x, err := strconv.Atoi(p)
		if err != nil {
			return n, false
		}
		n[i] = x
	}
	return n, true
}

// Newer indica si latest es una versión posterior a current. Versiones ilegibles: false.
func Newer(current, latest string) bool {
	c, ok1 := parse(current)
	l, ok2 := parse(latest)
	if !ok1 || !ok2 {
		return false
	}
	for i := range c {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func get(url string) (io.ReadCloser, error) {
	resp, err := Client.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	return resp.Body, nil
}

// Download baja la release para (goos, goarch), verifica su sha256 y deja el binario
// extraído en dir. Devuelve su ruta. No toca el binario en uso.
func Download(tag, goos, goarch, dir string) (string, error) {
	if !tagRe.MatchString(tag) {
		return "", fmt.Errorf("etiqueta de versión inválida %q", tag)
	}
	asset := fmt.Sprintf("v-ia_%s_%s.tar.gz", goos, goarch)
	base := fmt.Sprintf("%s/%s/releases/download/%s", DownloadBase, Repo, tag)

	sums, err := get(base + "/checksums.txt")
	if err != nil {
		return "", err
	}
	sb, _ := io.ReadAll(io.LimitReader(sums, 1<<20))
	sums.Close()
	if err := verifySums(sb, base); err != nil {
		return "", err
	}
	want := ""
	for _, l := range strings.Split(string(sb), "\n") {
		if f := strings.Fields(l); len(f) == 2 && f[1] == asset {
			if want != "" && want != f[0] {
				return "", fmt.Errorf("checksums.txt tiene hashes contradictorios para %s: descarga rechazada", asset)
			}
			want = f[0]
		}
	}
	if want == "" {
		return "", fmt.Errorf("checksums.txt no tiene hash para %s", asset)
	}

	body, err := get(base + "/" + asset)
	if err != nil {
		return "", err
	}
	defer body.Close()
	tgz := filepath.Join(dir, asset)
	f, err := os.Create(tgz)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(body, maxBin)); err != nil {
		f.Close()
		return "", err
	}
	f.Close()
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return "", fmt.Errorf("el hash no coincide (esperado %s, obtenido %s): descarga rechazada", want, got)
	}
	return extract(tgz, dir)
}

// extract saca solo la entrada llamada "v-ia" (ignora rutas: sin path traversal).
func extract(tgz, dir string) (string, error) {
	f, err := os.Open(tgz)
	if err != nil {
		return "", err
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	tr := tar.NewReader(zr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return "", errors.New("el archivo no contiene el binario v-ia")
		}
		if err != nil {
			return "", err
		}
		if h.Typeflag != tar.TypeReg || filepath.Base(h.Name) != "v-ia" {
			continue
		}
		out := filepath.Join(dir, "v-ia.new")
		o, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return "", err
		}
		n, err := io.Copy(o, io.LimitReader(tr, maxBin+1))
		o.Close()
		if err != nil || n > maxBin {
			os.Remove(out)
			return "", errors.New("binario ilegible o demasiado grande")
		}
		return out, nil
	}
}

var rename = os.Rename // sustituible en pruebas

// swap pone tmp en el lugar de self. Windows no deja sobrescribir un ejecutable en marcha, pero sí
// renombrarlo: se aparta como .old (se borra en la siguiente actualización) y, si algo falla, vuelve.
func swap(goos, tmp, self string) error {
	if goos != "windows" {
		return rename(tmp, self)
	}
	old := self + ".old"
	os.Remove(old)
	if err := rename(self, old); err != nil {
		return err
	}
	if err := rename(tmp, self); err != nil {
		if rerr := rename(old, self); rerr != nil {
			return fmt.Errorf("%w; tampoco pude restaurar el binario anterior (%v): está en %s, renómbralo a %s", err, rerr, old, self)
		}
		return err
	}
	os.Remove(old) // falla si sigue en ejecución
	return nil
}

// Replace sustituye self por newBin con un rename desde la misma carpeta (atómico en Unix; en Windows son dos pasos, ver swap).
func Replace(self, newBin string) error {
	if real, err := filepath.EvalSymlinks(self); err == nil {
		self = real
	}
	tmp := self + ".new"
	in, err := os.Open(newBin)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("no puedo escribir junto a %s: %w", self, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Sync(); err != nil { // a disco antes del rename: un corte no deja un binario vacío
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := swap(runtime.GOOS, tmp, self); err != nil {
		os.Remove(tmp)
		return err
	}
	if d, err := os.Open(filepath.Dir(self)); err == nil { // persiste también la entrada del directorio
		d.Sync()
		d.Close()
	}
	return nil
}

// RunGH ejecuta 'gh'; sustituible en pruebas.
var RunGH = func(args ...string) ([]byte, error) { return exec.Command("gh", args...).CombinedOutput() }

// HaveGH indica si la CLI de GitHub está disponible.
var HaveGH = func() bool { _, err := exec.LookPath("gh"); return err == nil }

// AssetPath es donde Download deja el tar.gz descargado.
func AssetPath(dir, goos, goarch string) string {
	return filepath.Join(dir, fmt.Sprintf("v-ia_%s_%s.tar.gz", goos, goarch))
}

// VerifyAttestation comprueba con 'gh attestation verify' que el archivo lo construyó el
// workflow de release de este repositorio (firma Sigstore), más fuerte que un hash servido
// desde el mismo origen. Requiere la CLI 'gh'.
func VerifyAttestation(file string) error {
	out, err := RunGH("attestation", "verify", file, "--repo", Repo)
	if err != nil {
		return fmt.Errorf("la atestación no es válida o no existe: %s", strings.TrimSpace(string(out)))
	}
	return nil
}
