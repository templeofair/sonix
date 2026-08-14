import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import {
  applyColourModeToImageData,
  colourModePreviewFilter,
  isColourMode,
  loadColourPrefs,
  migrateColourMode,
  preparePageBlob,
  saveColourPrefs,
  COLOUR_MODES,
} from './colourMode'
import {
  CLEAN_FLATTEN_STRENGTH,
  CLEAN_MAX_FLATTEN_STRENGTH,
  applyCleanGray,
  softFlattenGray,
} from './scanEnhance'

function makeImageData(w: number, h: number, fill: (x: number, y: number) => [number, number, number]): ImageData {
  const data = new Uint8ClampedArray(w * h * 4)
  for (let y = 0; y < h; y++) {
    for (let x = 0; x < w; x++) {
      const [r, g, b] = fill(x, y)
      const i = (y * w + x) * 4
      data[i] = r
      data[i + 1] = g
      data[i + 2] = b
      data[i + 3] = 255
    }
  }
  return { data, width: w, height: h, colorSpace: 'srgb' } as ImageData
}

describe('colour modes (Original | Clean only)', () => {
  it('exposes only original and clean', () => {
    expect([...COLOUR_MODES]).toEqual(['original', 'clean'])
    expect(isColourMode('original')).toBe(true)
    expect(isColourMode('clean')).toBe(true)
    expect(isColourMode('grayscale')).toBe(false)
    expect(isColourMode('bw')).toBe(false)
  })

  it('migrates legacy gray/bw to clean', () => {
    expect(migrateColourMode('grayscale')).toBe('clean')
    expect(migrateColourMode('bw')).toBe('clean')
    expect(migrateColourMode('clean')).toBe('clean')
  })

  it('original leaves pixels unchanged', () => {
    const img = makeImageData(1, 1, () => [10, 20, 30])
    applyColourModeToImageData(img, 'original')
    expect([...img.data]).toEqual([10, 20, 30, 255])
  })

  it('preview filter is none for original', () => {
    expect(colourModePreviewFilter('original')).toBe('none')
    expect(colourModePreviewFilter('clean')).toContain('grayscale')
  })
})

describe('Clean quality / regression guards', () => {
  it('keeps flatten strength under the safe ceiling', () => {
    expect(CLEAN_FLATTEN_STRENGTH).toBeLessThanOrEqual(CLEAN_MAX_FLATTEN_STRENGTH)
    expect(CLEAN_MAX_FLATTEN_STRENGTH).toBeLessThanOrEqual(0.5)
  })

  it('softFlatten clamps strength to the regression ceiling', () => {
    const w = 32
    const h = 32
    const gray = new Uint8Array(w * h).fill(180)
    const capped = softFlattenGray(gray, w, h, 1.0, 0.2)
    const atHalf = softFlattenGray(gray, w, h, 0.5, 0.2)
    // Asking for full Retinex must not exceed the 0.5 blend ceiling.
    expect([...capped]).toEqual([...atHalf])
  })

  it('Clean stays continuous-tone (not hard B&W)', () => {
    const img = makeImageData(48, 48, (x, y) => {
      const paper = 150 + Math.floor((x / 47) * 60)
      if (x === 24 && y > 8 && y < 40) return [30, 30, 30]
      return [paper, paper, paper]
    })
    applyCleanGray(img)
    const tones = new Set<number>()
    for (let i = 0; i < img.data.length; i += 4) tones.add(img.data[i])
    expect(tones.size).toBeGreaterThan(8)
    expect(tones.has(0) && tones.has(255) && tones.size <= 2).toBe(false)
  })

  it('Clean keeps ink darker than nearby paper', () => {
    const img = makeImageData(48, 48, (x) => {
      const paper = 140 + Math.floor((x / 47) * 70)
      if (x === 24) return [35, 35, 35]
      return [paper, paper, paper]
    })
    applyCleanGray(img)
    const ink = img.data[(24 * 48 + 24) * 4]
    const paper = img.data[(24 * 48 + 30) * 4]
    expect(ink).toBeLessThan(paper)
  })
})

describe('colour prefs', () => {
  beforeEach(() => localStorage.clear())
  afterEach(() => localStorage.clear())

  it('persists mode', () => {
    saveColourPrefs('clean')
    expect(loadColourPrefs()).toEqual({ mode: 'clean' })
  })

  it('loads legacy bw as clean', () => {
    localStorage.setItem('sonix.colourMode', 'bw')
    expect(loadColourPrefs().mode).toBe('clean')
  })
})

describe('preparePageBlob', () => {
  it('returns the same blob for original (deskew off by default)', async () => {
    const blob = new Blob([new Uint8Array([1, 2, 3, 4])], { type: 'image/jpeg' })
    expect(await preparePageBlob(blob, { mode: 'original' })).toBe(blob)
  })
})
