#!/usr/bin/env python3
"""Convierte docs/assets/logo-neon.png en PNGs pequeños (color real) para el banner de la TUI.

Cada celda de terminal = 1 px de ancho x 2 px de alto (medios bloques ▀▄█) con
color verdadero: se conserva el degradado blanco→azul y el neón violeta/azul.
Los píxeles oscuros quedan en negro puro y la TUI los dibuja como fondo.
Uso: python3 scripts/gen-banner.py   (requiere Pillow)
"""
from PIL import Image, ImageFilter

SRC = "docs/assets/logo-neon.png"
OUT = "internal/tui/assets/banner_{name}.png"
SIZES = {"large": 104, "small": 64}
GAMMA = 0.8   # aclara el neón fino que el reescalado atenúa
CUT = 38


def main():
    im = Image.open(SRC).convert("RGB")
    box = im.convert("L").point(lambda v: 255 if v > 40 else 0).getbbox()
    im = im.crop(box)
    w0, h0 = im.size
    for name, cols in SIZES.items():
        h = round(cols * h0 / w0)
        h += h % 2
        # MaxFilter antes de reducir: el contorno fino no se diluye
        k = 1
        src = im
        small = src.resize((cols, h), Image.LANCZOS)
        px = small.load()
        for y in range(h):
            for x in range(cols):
                r, g, b = px[x, y]
                r, g, b = (int(255 * (c / 255) ** GAMMA) for c in (r, g, b))
                px[x, y] = (0, 0, 0) if max(r, g, b) < CUT else (r, g, b)
        small.save(OUT.format(name=name), optimize=True)
        print(name, cols, "x", h, "px =", h // 2, "filas")


def kitty():
    """PNG RGBA grande para terminales con protocolo de gráficos kitty (Ghostty, kitty)."""
    im = Image.open(SRC).convert("RGB")
    im = im.crop(im.convert("L").point(lambda v: 255 if v > 40 else 0).getbbox())
    im = im.resize((900, round(900 * im.height / im.width)), Image.LANCZOS)
    a = Image.new("L", im.size)
    px, ap = im.load(), a.load()
    for y in range(im.height):
        for x in range(im.width):
            ap[x, y] = min(255, int(max(px[x, y]) * 1.6))
    im.putalpha(a)
    im.save("internal/tui/assets/banner_kitty.png", optimize=True)
    print("kitty", im.size)


main()
kitty()
