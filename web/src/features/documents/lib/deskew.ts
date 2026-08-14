/** Projection-profile deskew for slightly tilted phone captures. */

export type DeskewResult = {
  /** Degrees clockwise to straighten (negative = counter-clockwise). */
  angleDeg: number
  /** True when a rotation was applied (|angle| above threshold). */
  applied: boolean
}

const MIN_ABS_ANGLE = 0.35
const MAX_ABS_ANGLE = 7.5

function luminanceAt(data: Uint8ClampedArray, i: number): number {
  return (0.299 * data[i] + 0.587 * data[i + 1] + 0.114 * data[i + 2]) | 0
}

/** Score how “lined” an image is after a trial rotation (higher = sharper text rows). */
function projectionScore(data: Uint8ClampedArray, w: number, h: number, angleDeg: number): number {
  const rad = (angleDeg * Math.PI) / 180
  const cos = Math.cos(rad)
  const sin = Math.sin(rad)
  const cx = (w - 1) / 2
  const cy = (h - 1) / 2
  const rowInk = new Float64Array(h)
  // Sample a sparse grid for speed; enough for phone-letter deskew.
  const stepX = Math.max(1, Math.floor(w / 320))
  const stepY = Math.max(1, Math.floor(h / 480))
  for (let y = 0; y < h; y += stepY) {
    let ink = 0
    for (let x = 0; x < w; x += stepX) {
      const dx = x - cx
      const dy = y - cy
      const sx = Math.round(cx + dx * cos - dy * sin)
      const sy = Math.round(cy + dx * sin + dy * cos)
      if (sx < 0 || sy < 0 || sx >= w || sy >= h) continue
      const i = (sy * w + sx) * 4
      if (luminanceAt(data, i) < 140) ink++
    }
    rowInk[y] = ink
  }
  let sum = 0
  let sumSq = 0
  let n = 0
  for (let y = 0; y < h; y += stepY) {
    const v = rowInk[y]
    sum += v
    sumSq += v * v
    n++
  }
  if (n < 2) return 0
  const mean = sum / n
  return sumSq / n - mean * mean
}

/**
 * Estimate small skew via projection-profile variance over ±MAX_ABS_ANGLE.
 * Returns 0 when the signal is weak (blank page / photo of a wall).
 */
export function estimateSkewDegrees(imageData: ImageData): number {
  const { data, width: w, height: h } = imageData
  if (w < 32 || h < 32) return 0

  let bestAngle = 0
  let bestScore = -1
  // Coarse search then refine.
  for (let a = -MAX_ABS_ANGLE; a <= MAX_ABS_ANGLE + 1e-9; a += 1) {
    const s = projectionScore(data, w, h, a)
    if (s > bestScore) {
      bestScore = s
      bestAngle = a
    }
  }
  const lo = Math.max(-MAX_ABS_ANGLE, bestAngle - 1)
  const hi = Math.min(MAX_ABS_ANGLE, bestAngle + 1)
  for (let a = lo; a <= hi + 1e-9; a += 0.25) {
    const s = projectionScore(data, w, h, a)
    if (s > bestScore) {
      bestScore = s
      bestAngle = a
    }
  }
  if (Math.abs(bestAngle) < MIN_ABS_ANGLE) return 0
  return Math.round(bestAngle * 100) / 100
}

/**
 * Rotate ImageData by angleDeg (clockwise positive) around centre.
 * Output canvas is expanded to fit; corners filled white.
 */
export function rotateImageData(src: ImageData, angleDeg: number): ImageData {
  if (Math.abs(angleDeg) < 1e-6) return src
  const rad = (angleDeg * Math.PI) / 180
  const cos = Math.cos(rad)
  const sin = Math.sin(rad)
  const w = src.width
  const h = src.height
  const cx = (w - 1) / 2
  const cy = (h - 1) / 2
  const corners = [
    [0, 0],
    [w, 0],
    [w, h],
    [0, h],
  ].map(([x, y]) => {
    const dx = x - cx
    const dy = y - cy
    return [cx + dx * cos - dy * sin, cy + dx * sin + dy * cos]
  })
  let minX = Infinity
  let minY = Infinity
  let maxX = -Infinity
  let maxY = -Infinity
  for (const [x, y] of corners) {
    if (x < minX) minX = x
    if (y < minY) minY = y
    if (x > maxX) maxX = x
    if (y > maxY) maxY = y
  }
  const ow = Math.max(1, Math.ceil(maxX - minX))
  const oh = Math.max(1, Math.ceil(maxY - minY))
  const out = new Uint8ClampedArray(ow * oh * 4)
  // White background.
  for (let i = 0; i < out.length; i += 4) {
    out[i] = 255
    out[i + 1] = 255
    out[i + 2] = 255
    out[i + 3] = 255
  }
  const sdata = src.data
  const invCos = Math.cos(-rad)
  const invSin = Math.sin(-rad)
  const ocx = (ow - 1) / 2
  const ocy = (oh - 1) / 2
  for (let y = 0; y < oh; y++) {
    for (let x = 0; x < ow; x++) {
      const dx = x - ocx
      const dy = y - ocy
      const sx = Math.round(cx + dx * invCos - dy * invSin)
      const sy = Math.round(cy + dx * invSin + dy * invCos)
      if (sx < 0 || sy < 0 || sx >= w || sy >= h) continue
      const si = (sy * w + sx) * 4
      const di = (y * ow + x) * 4
      out[di] = sdata[si]
      out[di + 1] = sdata[si + 1]
      out[di + 2] = sdata[si + 2]
      out[di + 3] = sdata[si + 3]
    }
  }
  if (typeof ImageData !== 'undefined') {
    try {
      return new ImageData(out, ow, oh)
    } catch {
      /* fall through */
    }
  }
  return { width: ow, height: oh, data: out } as ImageData
}

/**
 * Deskew in place when a small tilt is detected. Mutates by replacing pixels
 * via returned ImageData (caller should use the return value).
 */
export function deskewImageData(imageData: ImageData): { imageData: ImageData } & DeskewResult {
  const angleDeg = estimateSkewDegrees(imageData)
  if (Math.abs(angleDeg) < MIN_ABS_ANGLE) {
    return { imageData, angleDeg: 0, applied: false }
  }
  return { imageData: rotateImageData(imageData, -angleDeg), angleDeg, applied: true }
}
