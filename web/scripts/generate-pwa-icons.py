#!/usr/bin/env python3
"""Generate simple maskable PNGs for the Sonix PWA (no Pillow required)."""

from __future__ import annotations

import struct
import zlib
from pathlib import Path

# Brand accent #5b7c99
ACCENT = (0x5B, 0x7C, 0x99, 0xFF)
WHITE = (0xFF, 0xFF, 0xFF, 0xFF)


def png_chunk(tag: bytes, data: bytes) -> bytes:
    return struct.pack(">I", len(data)) + tag + data + struct.pack(">I", zlib.crc32(tag + data) & 0xFFFFFFFF)


def write_png(path: Path, size: int, rgba: list[tuple[int, int, int, int]]) -> None:
    raw = bytearray()
    for y in range(size):
        raw.append(0)  # filter none
        for x in range(size):
            raw.extend(rgba[y * size + x])
    ihdr = struct.pack(">IIBBBBB", size, size, 8, 6, 0, 0, 0)
    data = b"".join(
        [
            png_chunk(b"IHDR", ihdr),
            png_chunk(b"IDAT", zlib.compress(bytes(raw), 9)),
            png_chunk(b"IEND", b""),
        ]
    )
    path.write_bytes(b"\x89PNG\r\n\x1a\n" + data)


def filled(size: int, color: tuple[int, int, int, int]) -> list[tuple[int, int, int, int]]:
    return [color] * (size * size)


def draw_doc_icon(size: int) -> list[tuple[int, int, int, int]]:
    """Accent fill with a simple white document glyph (maskable-safe inset)."""
    px = filled(size, ACCENT)
    # Safe zone ~80% for maskable icons
    margin = int(size * 0.18)
    left, right = margin, size - margin
    top, bottom = margin, size - margin
    fold = int(size * 0.12)

    def set_px(x: int, y: int, c: tuple[int, int, int, int] = WHITE) -> None:
        if 0 <= x < size and 0 <= y < size:
            px[y * size + x] = c

    # Document body
    for y in range(top, bottom):
        for x in range(left, right - fold):
            set_px(x, y)
    # Folded corner cut
    for y in range(top, top + fold):
        for x in range(right - fold, right):
            if (x - (right - fold)) + (y - top) >= fold:
                continue
            set_px(x, y)
    # Text lines
    line_left = left + int(size * 0.08)
    line_right = right - fold - int(size * 0.08)
    for i, y_frac in enumerate((0.38, 0.50, 0.62)):
        y = int(size * y_frac)
        end = line_right if i < 2 else line_left + int((line_right - line_left) * 0.65)
        for x in range(line_left, end):
            set_px(x, y, ACCENT)
            if y + 1 < size:
                set_px(x, y + 1, ACCENT)
    return px


def main() -> None:
    out = Path(__file__).resolve().parents[1] / "public" / "icons"
    out.mkdir(parents=True, exist_ok=True)
    for size, name in ((192, "icon-192.png"), (512, "icon-512.png"), (180, "apple-touch-icon.png")):
        write_png(out / name, size, draw_doc_icon(size))
        print(f"wrote {out / name}")


if __name__ == "__main__":
    main()
