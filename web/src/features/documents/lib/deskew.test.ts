import { describe, it, expect } from 'vitest'
import { estimateSkewDegrees, rotateImageData } from './deskew'

function makeImageData(w: number, h: number, fill: (x: number, y: number) => number): ImageData {
  const data = new Uint8ClampedArray(w * h * 4)
  for (let y = 0; y < h; y++) {
    for (let x = 0; x < w; x++) {
      const v = fill(x, y)
      const i = (y * w + x) * 4
      data[i] = v
      data[i + 1] = v
      data[i + 2] = v
      data[i + 3] = 255
    }
  }
  return { data, width: w, height: h, colorSpace: 'srgb' } as ImageData
}

describe('deskew', () => {
  it('returns ~0 for an already-level text-like page', () => {
    // Horizontal dark bands every 8 rows on white.
    const img = makeImageData(80, 120, (_x, y) => (y % 8 < 2 ? 20 : 240))
    expect(Math.abs(estimateSkewDegrees(img))).toBeLessThan(0.5)
  })

  it('rotateImageData expands canvas and fills corners white', () => {
    const img = makeImageData(20, 10, () => 0)
    const out = rotateImageData(img, 15)
    expect(out.width).toBeGreaterThanOrEqual(20)
    expect(out.height).toBeGreaterThanOrEqual(10)
    // Corner should be paper white fill.
    expect(out.data[0]).toBe(255)
  })
})
