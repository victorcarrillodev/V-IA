#!/usr/bin/env python3
"""Convierte docs/assets/logo.png en mapas de píxeles para el banner de la TUI.

Cada celda de terminal = 1 px de ancho x 2 px de alto (medios bloques ▀▄█).
Clases: '.' fondo, 'W' blanco, 'P' violeta. Sin dependencias salvo Pillow.
Uso: python3 scripts/gen-banner.py
"""
from PIL import Image

SRC = "docs/assets/logo.png"
OUT = "internal/tui/assets/banner_{name}.txt"
SIZES = {"large": 76, "small": 46}  # ancho en columnas


def classify(r, g, b):
    if b > 105 and b - g > 55 and r < 150:
        return "P"
    if min(r, g, b) > 150:
        return "W"
    return "."


def main():
    im = Image.open(SRC).convert("RGB")
    # recorte al contenido (todo lo que no es casi negro)
    bw = im.convert("L").point(lambda v: 255 if v > 40 else 0)
    box = bw.getbbox()
    im = im.crop(box)
    w0, h0 = im.size
    for name, cols in SIZES.items():
        h = round(cols * h0 / w0)
        h += h % 2  # par: dos filas de píxel por fila de terminal
        small = im.resize((cols, h), Image.LANCZOS)
        rows = ["".join(classify(*small.getpixel((x, y))) for x in range(cols)) for y in range(h)]
        with open(OUT.format(name=name), "w") as f:
            f.write("\n".join(rows) + "\n")
        print(name, cols, "x", h, "px =", h // 2, "filas de terminal")
        print("\n".join(rows))


main()
