#!/bin/sh
# Instala el binario v-ia desde la última release de GitHub. No pide sudo.
# Uso: curl -fsSL https://raw.githubusercontent.com/victorcarrillodev/V-IA/main/install.sh | sh
set -eu
REPO="victorcarrillodev/V-IA"
DEST="${V_IA_DIR:-$HOME/.local/bin}"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "arquitectura no soportada: $(uname -m)" >&2; exit 1 ;;
esac
case "$os" in linux|darwin) ;; *) echo "sistema no soportado: $os" >&2; exit 1 ;; esac

base="https://github.com/$REPO/releases/latest/download"
asset="v-ia_${os}_${arch}.tar.gz"
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
echo "Descargando $base/$asset"
curl -fsSL "$base/$asset" -o "$tmp/$asset"
curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt"

# integridad: el hash de la descarga debe coincidir con checksums.txt
want=$(awk -v f="$asset" '$2 == f {print $1}' "$tmp/checksums.txt")
[ -n "$want" ] || { echo "no hay hash para $asset en checksums.txt" >&2; exit 1; }
if command -v sha256sum >/dev/null 2>&1; then got=$(sha256sum "$tmp/$asset" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then got=$(shasum -a 256 "$tmp/$asset" | awk '{print $1}')
else echo "falta sha256sum o shasum para verificar la descarga" >&2; exit 1; fi
[ "$got" = "$want" ] || { echo "¡el hash no coincide! esperado $want, obtenido $got" >&2; exit 1; }
echo "✓ integridad verificada (sha256)"

tar -xzf "$tmp/$asset" -C "$tmp"
mkdir -p "$DEST"
install -m 0755 "$tmp/v-ia" "$DEST/v-ia"
echo "✓ v-ia instalado en $DEST/v-ia"
case ":$PATH:" in
  *":$DEST:"*) echo "Listo. Ejecuta:  v-ia" ;;
  *) echo "Añade $DEST a tu PATH (p. ej. export PATH=\"$DEST:\$PATH\" en ~/.zshrc) y ejecuta:  v-ia" ;;
esac
