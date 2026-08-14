import { describe, it, expect } from 'vitest'
import {
  CLEAN_FLATTEN_STRENGTH,
  CLEAN_KERNEL_FRACTION,
  CLEAN_SHARPEN_AMOUNT,
  gentlePaperBoost,
  illuminationKernel,
  unsharpGray,
} from './scanEnhance'

describe('scanEnhance Clean helpers', () => {
  it('illumination kernel is odd and bounded', () => {
    const k = illuminationKernel(1000, CLEAN_KERNEL_FRACTION, 51)
    expect(k % 2).toBe(1)
    expect(k).toBeGreaterThanOrEqual(51)
  })

  it('paper boost protects dark ink', () => {
    const gray = Uint8Array.from([40, 120, 200])
    const out = gentlePaperBoost(gray)
    expect(out[0]).toBe(40)
    expect(out[2]).toBeGreaterThan(200)
  })

  it('unsharp amount stays mild', () => {
    expect(CLEAN_SHARPEN_AMOUNT).toBeLessThanOrEqual(0.35)
    expect(CLEAN_FLATTEN_STRENGTH).toBeLessThanOrEqual(0.5)
    const w = 5
    const h = 5
    const gray = new Uint8Array(w * h).fill(128)
    gray[12] = 200
    const out = unsharpGray(gray, w, h, 1.0) // request high — function clamps
    // Centre should move, but not explode past 255.
    expect(out[12]).toBeLessThanOrEqual(255)
    expect(out[12]).toBeGreaterThanOrEqual(200)
  })
})
