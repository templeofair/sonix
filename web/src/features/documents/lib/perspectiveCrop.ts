import { deskewImageData } from './deskew'
import { applyColourModeToImageData, COLOUR_JPEG_QUALITY, type ColourMode } from './colourMode'

/** Image-space point (pixels). */
export type Point = { x: number; y: number }

/** Corners in order: top-left, top-right, bottom-right, bottom-left. */
export type Quad = [Point, Point, Point, Point]

export const CROP_JPEG_QUALITY = COLOUR_JPEG_QUALITY
/** Cap long edge of crop working / output canvases (full-res apply). */
export const CROP_MAX_LONG_EDGE = 3500
/** Preview canvas long-edge cap while dragging. */
export const CROP_PREVIEW_MAX_EDGE = 1280

export function dist(a: Point, b: Point): number {
  const dx = a.x - b.x
  const dy = a.y - b.y
  return Math.hypot(dx, dy)
}

/** Slightly inset full-page quad so handles are visible on the first open. */
export function defaultQuad(width: number, height: number, insetRatio = 0.04): Quad {
  const ix = Math.max(8, width * insetRatio)
  const iy = Math.max(8, height * insetRatio)
  return [
    { x: ix, y: iy },
    { x: width - ix, y: iy },
    { x: width - ix, y: height - iy },
    { x: ix, y: height - iy },
  ]
}

export function fullPageQuad(width: number, height: number): Quad {
  return [
    { x: 0, y: 0 },
    { x: width, y: 0 },
    { x: width, y: height },
    { x: 0, y: height },
  ]
}

export function clampPoint(p: Point, width: number, height: number): Point {
  return {
    x: Math.min(width, Math.max(0, p.x)),
    y: Math.min(height, Math.max(0, p.y)),
  }
}

export function outputSizeForQuad(quad: Quad): { width: number; height: number } {
  const [tl, tr, br, bl] = quad
  const top = dist(tl, tr)
  const bottom = dist(bl, br)
  const left = dist(tl, bl)
  const right = dist(tr, br)
  let width = Math.max(top, bottom)
  let height = Math.max(left, right)
  const long = Math.max(width, height)
  if (long > CROP_MAX_LONG_EDGE) {
    const s = CROP_MAX_LONG_EDGE / long
    width *= s
    height *= s
  }
  return {
    width: Math.max(1, Math.round(width)),
    height: Math.max(1, Math.round(height)),
  }
}

/**
 * Solve for perspective matrix mapping unit square (0,0)-(1,0)-(1,1)-(0,1)
 * to the given quad (TL,TR,BR,BL). Returns 8 coeffs of the 3x3 matrix
 * (last element fixed to 1): [a,b,c, d,e,f, g,h].
 */
export function getPerspectiveCoeffs(quad: Quad): Float64Array {
  const [[x0, y0], [x1, y1], [x2, y2], [x3, y3]] = [
    [quad[0].x, quad[0].y],
    [quad[1].x, quad[1].y],
    [quad[2].x, quad[2].y],
    [quad[3].x, quad[3].y],
  ]

  // Map unit square corners → quad via bilinear + perspective terms.
  // Standard CV approach: solve 8x8 for homography from dst rect to src quad
  // is done in warp; here we compute coeffs for (u,v) in [0,1]² → (x,y).
  const dx1 = x1 - x2
  const dx2 = x3 - x2
  const dx3 = x0 - x1 + x2 - x3
  const dy1 = y1 - y2
  const dy2 = y3 - y2
  const dy3 = y0 - y1 + y2 - y3

  let g = 0
  let h = 0
  const den = dx1 * dy2 - dy1 * dx2
  if (Math.abs(den) > 1e-8) {
    g = (dx3 * dy2 - dy3 * dx2) / den
    h = (dx1 * dy3 - dy1 * dx3) / den
  }

  const a = x1 - x0 + g * x1
  const b = x3 - x0 + h * x3
  const c = x0
  const d = y1 - y0 + g * y1
  const e = y3 - y0 + h * y3
  const f = y0

  return new Float64Array([a, b, c, d, e, f, g, h])
}

/** Map normalized (u,v) in [0,1]² through perspective coeffs to image point. */
export function mapUnitToImage(coeffs: Float64Array, u: number, v: number): Point {
  const [a, b, c, d, e, f, g, h] = coeffs
  const den = g * u + h * v + 1
  return {
    x: (a * u + b * v + c) / den,
    y: (d * u + e * v + f) / den,
  }
}

function sampleBilinear(
  data: Uint8ClampedArray,
  w: number,
  h: number,
  x: number,
  y: number
): [number, number, number, number] {
  if (x < 0 || y < 0 || x >= w - 1 || y >= h - 1) {
    const xi = Math.min(w - 1, Math.max(0, Math.round(x)))
    const yi = Math.min(h - 1, Math.max(0, Math.round(y)))
    const i = (yi * w + xi) * 4
    return [data[i], data[i + 1], data[i + 2], data[i + 3]]
  }
  const x0 = Math.floor(x)
  const y0 = Math.floor(y)
  const x1 = x0 + 1
  const y1 = y0 + 1
  const fx = x - x0
  const fy = y - y0
  const i00 = (y0 * w + x0) * 4
  const i10 = (y0 * w + x1) * 4
  const i01 = (y1 * w + x0) * 4
  const i11 = (y1 * w + x1) * 4
  const out: [number, number, number, number] = [0, 0, 0, 0]
  for (let c = 0; c < 4; c++) {
    const v00 = data[i00 + c]
    const v10 = data[i10 + c]
    const v01 = data[i01 + c]
    const v11 = data[i11 + c]
    const v0 = v00 * (1 - fx) + v10 * fx
    const v1 = v01 * (1 - fx) + v11 * fx
    out[c] = Math.round(v0 * (1 - fy) + v1 * fy)
  }
  return out
}

/** Warp source ImageData through quad → axis-aligned output. */
export function warpPerspective(
  src: ImageData,
  quad: Quad,
  outWidth: number,
  outHeight: number
): ImageData {
  const coeffs = getPerspectiveCoeffs(quad)
  const outData = new Uint8ClampedArray(outWidth * outHeight * 4)
  const sw = src.width
  const sh = src.height
  const sdata = src.data

  for (let y = 0; y < outHeight; y++) {
    const v = outHeight <= 1 ? 0 : y / (outHeight - 1)
    for (let x = 0; x < outWidth; x++) {
      const u = outWidth <= 1 ? 0 : x / (outWidth - 1)
      const p = mapUnitToImage(coeffs, u, v)
      const [r, g, b, a] = sampleBilinear(sdata, sw, sh, p.x, p.y)
      const i = (y * outWidth + x) * 4
      outData[i] = r
      outData[i + 1] = g
      outData[i + 2] = b
      outData[i + 3] = a
    }
  }
  if (typeof ImageData !== 'undefined') {
    try {
      return new ImageData(outData, outWidth, outHeight)
    } catch {
      /* fall through for odd environments */
    }
  }
  return { width: outWidth, height: outHeight, data: outData } as ImageData
}

async function blobToImageData(blob: Blob, maxLongEdge: number): Promise<ImageData> {
  const bitmap = await createImageBitmap(blob)
  try {
    let tw = bitmap.width
    let th = bitmap.height
    const long = Math.max(tw, th)
    if (long > maxLongEdge) {
      const s = maxLongEdge / long
      tw = Math.max(1, Math.round(tw * s))
      th = Math.max(1, Math.round(th * s))
    }
    const canvas = document.createElement('canvas')
    canvas.width = tw
    canvas.height = th
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('Could not read image for crop')
    ctx.drawImage(bitmap, 0, 0, tw, th)
    const data = ctx.getImageData(0, 0, tw, th)
    canvas.width = 0
    canvas.height = 0
    return data
  } finally {
    bitmap.close()
  }
}

function imageDataToBlob(data: ImageData, quality = CROP_JPEG_QUALITY): Promise<Blob> {
  const canvas = document.createElement('canvas')
  canvas.width = data.width
  canvas.height = data.height
  const ctx = canvas.getContext('2d')
  if (!ctx) return Promise.reject(new Error('Could not encode crop'))
  ctx.putImageData(data, 0, 0)
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (b) => {
        canvas.width = 0
        canvas.height = 0
        if (b) resolve(b)
        else reject(new Error('Could not encode crop'))
      },
      'image/jpeg',
      quality
    )
  })
}

/**
 * Apply perspective crop at up to CROP_MAX_LONG_EDGE.
 * Near-full-page quads skip the warp and return the original blob when nothing
 * else changes, so we do not stack a useless JPEG encode.
 * Optional colour mode is applied in the same pass (one encode).
 * Auto-deskew is **off** by default (hurts quality when it false-positives).
 */
export async function perspectiveCropBlob(
  blob: Blob,
  quad: Quad,
  opts?: {
    maxLongEdge?: number
    quality?: number
    colourMode?: ColourMode
    deskew?: boolean
  }
): Promise<{ blob: Blob; width: number; height: number }> {
  const maxLong = opts?.maxLongEdge ?? CROP_MAX_LONG_EDGE
  const quality = opts?.quality ?? CROP_JPEG_QUALITY
  const colourMode = opts?.colourMode ?? 'original'
  const doDeskew = opts?.deskew === true

  const bitmap = await createImageBitmap(blob)
  const natW = bitmap.width
  const natH = bitmap.height
  bitmap.close()

  const nearFull = isNearFullPage(quad, natW, natH)
  let working: ImageData

  if (nearFull && natW <= maxLong && natH <= maxLong) {
    working = await blobToImageData(blob, maxLong)
  } else {
    const src = await blobToImageData(blob, maxLong)
    const scaleX = src.width / natW
    const scaleY = src.height / natH
    const scaled: Quad = [
      { x: quad[0].x * scaleX, y: quad[0].y * scaleY },
      { x: quad[1].x * scaleX, y: quad[1].y * scaleY },
      { x: quad[2].x * scaleX, y: quad[2].y * scaleY },
      { x: quad[3].x * scaleX, y: quad[3].y * scaleY },
    ]
    if (nearFull) {
      working = src
    } else {
      const size = outputSizeForQuad(scaled)
      working = warpPerspective(src, scaled, size.width, size.height)
    }
  }

  let changed = !nearFull || working.width !== natW || working.height !== natH
  if (doDeskew) {
    const d = deskewImageData(working)
    working = d.imageData
    if (d.applied) changed = true
  }
  if (colourMode !== 'original') {
    applyColourModeToImageData(working, colourMode)
    changed = true
  }

  if (!changed) {
    return { blob, width: natW, height: natH }
  }
  const out = await imageDataToBlob(working, quality)
  return { blob: out, width: working.width, height: working.height }
}

/** True when quad is (near) the full image rectangle — skip path can stay byte-identical only if we never re-encode; callers should skip calling crop. */
export function isNearFullPage(quad: Quad, width: number, height: number, tol = 2): boolean {
  const full = fullPageQuad(width, height)
  return quad.every((p, i) => Math.abs(p.x - full[i].x) <= tol && Math.abs(p.y - full[i].y) <= tol)
}
