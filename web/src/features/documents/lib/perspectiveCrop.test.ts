import { describe, it, expect } from 'vitest'
import {
  defaultQuad,
  dist,
  fullPageQuad,
  getPerspectiveCoeffs,
  isNearFullPage,
  mapUnitToImage,
  outputSizeForQuad,
  warpPerspective,
} from './perspectiveCrop'

describe('perspectiveCrop helpers', () => {
  it('defaultQuad is inset from full page', () => {
    const q = defaultQuad(1000, 800)
    expect(q[0].x).toBeGreaterThan(0)
    expect(q[2].x).toBeLessThan(1000)
    expect(isNearFullPage(fullPageQuad(1000, 800), 1000, 800)).toBe(true)
    expect(isNearFullPage(q, 1000, 800)).toBe(false)
  })

  it('outputSizeForQuad uses edge lengths', () => {
    const q = fullPageQuad(200, 100)
    const size = outputSizeForQuad(q)
    expect(size.width).toBe(200)
    expect(size.height).toBe(100)
  })

  it('perspective coeffs map unit corners to quad', () => {
    const quad = fullPageQuad(100, 50)
    const c = getPerspectiveCoeffs(quad)
    const tl = mapUnitToImage(c, 0, 0)
    const br = mapUnitToImage(c, 1, 1)
    expect(tl.x).toBeCloseTo(0, 5)
    expect(tl.y).toBeCloseTo(0, 5)
    expect(br.x).toBeCloseTo(100, 5)
    expect(br.y).toBeCloseTo(50, 5)
  })

  it('warpPerspective produces requested dimensions', () => {
    const w = 40
    const h = 30
    const data = new Uint8ClampedArray(w * h * 4)
    for (let i = 0; i < data.length; i += 4) {
      data[i] = 200
      data[i + 1] = 100
      data[i + 2] = 50
      data[i + 3] = 255
    }
    const src = { width: w, height: h, data } as ImageData
    const out = warpPerspective(src, fullPageQuad(w, h), 20, 15)
    expect(out.width).toBe(20)
    expect(out.height).toBe(15)
    expect(out.data[0]).toBeGreaterThan(150)
  })

  it('dist is symmetric', () => {
    expect(dist({ x: 0, y: 0 }, { x: 3, y: 4 })).toBe(5)
  })
})
