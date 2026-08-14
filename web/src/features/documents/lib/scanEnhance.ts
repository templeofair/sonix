/**
 * Clean-mode document enhancement (MakeACopy-inspired, pure ImageData).
 *
 * Regression rules (do not break these):
 * - Never auto-deskew here (false tilts + NN rotate destroy quality).
 * - Never hard-binarize (0/255) — that speckles phone JPEGs.
 * - Never full Retinex (strength 1.0) — amplifies JPEG grain.
 * - Soft flatten + gentle paper lift + mild unsharp only.
 */

/** Tunables — keep conservative; tests lock the “not binary / not full Retinex” contract. */
export const CLEAN_FLATTEN_STRENGTH = 0.4
export const CLEAN_KERNEL_FRACTION = 0.14
export const CLEAN_SHARPEN_AMOUNT = 0.2
/** Max flatten strength Clean is allowed to use (regression ceiling). */
export const CLEAN_MAX_FLATTEN_STRENGTH = 0.5

/** Odd kernel size from a fraction of the short edge (min floor on large pages). */
export function illuminationKernel(shortEdge: number, fraction = CLEAN_KERNEL_FRACTION, minSize = 51): number {
  if (shortEdge < 3) return 3
  let k = Math.floor(shortEdge * fraction)
  if (shortEdge >= minSize) k = Math.max(minSize, k)
  k = Math.max(3, k)
  const maxOdd = shortEdge % 2 === 1 ? shortEdge : shortEdge - 1
  k = Math.min(k, Math.max(3, maxOdd))
  if (k % 2 === 0) k++
  if (k > maxOdd) k = maxOdd
  return k
}

function buildIntegral(gray: Uint8Array, w: number, h: number): Float64Array {
  const sum = new Float64Array((w + 1) * (h + 1))
  for (let y = 1; y <= h; y++) {
    let row = 0
    for (let x = 1; x <= w; x++) {
      row += gray[(y - 1) * w + (x - 1)]
      sum[y * (w + 1) + x] = sum[(y - 1) * (w + 1) + x] + row
    }
  }
  return sum
}

function rectMean(sum: Float64Array, w1: number, x0: number, y0: number, x1: number, y1: number): number {
  const n = (x1 - x0) * (y1 - y0)
  if (n <= 0) return 0
  const A = sum[y0 * w1 + x0]
  const B = sum[y0 * w1 + x1]
  const C = sum[y1 * w1 + x0]
  const D = sum[y1 * w1 + x1]
  return (D - B - C + A) / n
}

/** Box-blur approximation of a large Gaussian illumination field. */
export function boxBlurGray(gray: Uint8Array, w: number, h: number, kernel: number): Uint8Array {
  const k = kernel % 2 === 1 ? kernel : kernel + 1
  const half = Math.floor(k / 2)
  const sum = buildIntegral(gray, w, h)
  const w1 = w + 1
  const out = new Uint8Array(w * h)
  for (let y = 0; y < h; y++) {
    const y0 = Math.max(0, y - half)
    const y1 = Math.min(h, y + half + 1)
    for (let x = 0; x < w; x++) {
      const x0 = Math.max(0, x - half)
      const x1 = Math.min(w, x + half + 1)
      out[y * w + x] = Math.min(255, Math.max(0, Math.round(rectMean(sum, w1, x0, y0, x1, y1))))
    }
  }
  return out
}

/** Divide gray by a blurred copy. Prefer {@link softFlattenGray} for Clean. */
export function backgroundDivideGray(
  gray: Uint8Array,
  w: number,
  h: number,
  kernelFraction = CLEAN_KERNEL_FRACTION
): Uint8Array {
  const k = illuminationKernel(Math.min(w, h), kernelFraction)
  const bg = boxBlurGray(gray, w, h, k)
  const out = new Uint8Array(w * h)
  for (let i = 0; i < out.length; i++) {
    const b = Math.max(1, bg[i])
    out[i] = Math.min(255, Math.max(0, Math.round((gray[i] / b) * 255)))
  }
  return out
}

/**
 * Blend original gray with a flattened copy.
 * `strength` 0 = original, 1 = full divide (unsafe on phone JPEGs).
 */
export function softFlattenGray(
  gray: Uint8Array,
  w: number,
  h: number,
  strength = CLEAN_FLATTEN_STRENGTH,
  kernelFraction = CLEAN_KERNEL_FRACTION
): Uint8Array {
  const a = Math.min(CLEAN_MAX_FLATTEN_STRENGTH, Math.max(0, strength))
  const flat = backgroundDivideGray(gray, w, h, kernelFraction)
  const out = new Uint8Array(w * h)
  for (let i = 0; i < out.length; i++) {
    out[i] = Math.min(255, Math.max(0, Math.round(gray[i] * (1 - a) + flat[i] * a)))
  }
  return out
}

export function rgbaToGray(data: Uint8ClampedArray, w: number, h: number): Uint8Array {
  const gray = new Uint8Array(w * h)
  for (let i = 0, p = 0; i < data.length; i += 4, p++) {
    gray[p] = (0.299 * data[i] + 0.587 * data[i + 1] + 0.114 * data[i + 2]) | 0
  }
  return gray
}

export function writeGrayToRgba(gray: Uint8Array, data: Uint8ClampedArray): void {
  for (let p = 0, i = 0; p < gray.length; p++, i += 4) {
    const v = gray[p]
    data[i] = v
    data[i + 1] = v
    data[i + 2] = v
  }
}

/** 3×3 mean blur (copy-safe). */
export function blurGray3x3(gray: Uint8Array, w: number, h: number): Uint8Array {
  const out = new Uint8Array(w * h)
  for (let y = 0; y < h; y++) {
    for (let x = 0; x < w; x++) {
      let s = 0
      let n = 0
      for (let dy = -1; dy <= 1; dy++) {
        const yy = y + dy
        if (yy < 0 || yy >= h) continue
        for (let dx = -1; dx <= 1; dx++) {
          const xx = x + dx
          if (xx < 0 || xx >= w) continue
          s += gray[yy * w + xx]
          n++
        }
      }
      out[y * w + x] = (s / n) | 0
    }
  }
  return out
}

/**
 * Lift paper tones toward white without touching dark ink.
 * Uses a smooth ramp above ~100 so umlaut dots / thin strokes stay.
 */
export function gentlePaperBoost(gray: Uint8Array): Uint8Array {
  const out = new Uint8Array(gray.length)
  for (let i = 0; i < gray.length; i++) {
    const v = gray[i]
    if (v < 100) {
      out[i] = v
      continue
    }
    const t = (v - 100) / 155
    out[i] = Math.min(255, Math.round(v + t * t * 22))
  }
  return out
}

/** Mild unsharp mask — amount stays low so JPEG grain is not amplified. */
export function unsharpGray(
  gray: Uint8Array,
  w: number,
  h: number,
  amount = CLEAN_SHARPEN_AMOUNT
): Uint8Array {
  const a = Math.min(0.35, Math.max(0, amount))
  const blurred = blurGray3x3(gray, w, h)
  const out = new Uint8Array(gray.length)
  for (let i = 0; i < gray.length; i++) {
    out[i] = Math.min(255, Math.max(0, Math.round(gray[i] + a * (gray[i] - blurred[i]))))
  }
  return out
}

/**
 * Clean: luminance → soft shadow flatten → gentle paper boost → mild sharpen.
 * Output stays continuous-tone grayscale (never binary).
 */
export function applyCleanGray(imageData: ImageData): void {
  const { data, width: w, height: h } = imageData
  const gray = rgbaToGray(data, w, h)
  let work = softFlattenGray(gray, w, h, CLEAN_FLATTEN_STRENGTH, CLEAN_KERNEL_FRACTION)
  work = gentlePaperBoost(work)
  work = unsharpGray(work, w, h, CLEAN_SHARPEN_AMOUNT)
  writeGrayToRgba(work, data)
}
