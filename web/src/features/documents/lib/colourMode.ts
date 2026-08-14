/** Capture / review colour modes: Original (passthrough) or Clean (scanner-like gray). */

import { deskewImageData } from './deskew'
import { applyCleanGray } from './scanEnhance'

export type ColourMode = 'original' | 'clean'

export const COLOUR_MODES: readonly ColourMode[] = ['original', 'clean'] as const

export const COLOUR_MODE_LABELS: Record<ColourMode, string> = {
  original: 'Original',
  clean: 'Clean',
}

export const COLOUR_MODE_STORAGE_KEY = 'sonix.colourMode'

/** JPEG quality for Clean encode (high — avoid stacking visible loss). */
export const COLOUR_JPEG_QUALITY = 0.96

export function isColourMode(value: unknown): value is ColourMode {
  return value === 'original' || value === 'clean'
}

/** Map legacy Gray / B&W prefs onto Clean (the remaining refinement mode). */
export function migrateColourMode(value: unknown): ColourMode | null {
  if (isColourMode(value)) return value
  if (value === 'grayscale' || value === 'bw') return 'clean'
  return null
}

export function loadColourPrefs(): { mode: ColourMode } {
  let mode: ColourMode = 'original'
  try {
    const storedMode = localStorage.getItem(COLOUR_MODE_STORAGE_KEY)
    const migrated = migrateColourMode(storedMode)
    if (migrated) mode = migrated
  } catch {
    /* private mode / unavailable */
  }
  return { mode }
}

export function saveColourPrefs(mode: ColourMode): void {
  try {
    localStorage.setItem(COLOUR_MODE_STORAGE_KEY, mode)
  } catch {
    /* ignore */
  }
}

/** CSS filter for approximate live preview (capture still uses real Clean). */
export function colourModePreviewFilter(mode: ColourMode): string {
  if (mode === 'clean') return 'grayscale(1) contrast(1.1) brightness(1.04)'
  return 'none'
}

/** Mutates ImageData in place. */
export function applyColourModeToImageData(imageData: ImageData, mode: ColourMode): void {
  if (mode === 'original') return
  applyCleanGray(imageData)
}

function imageDataToJpegBlob(data: ImageData, quality = COLOUR_JPEG_QUALITY): Promise<Blob> {
  const canvas = document.createElement('canvas')
  canvas.width = data.width
  canvas.height = data.height
  const ctx = canvas.getContext('2d')
  if (!ctx) return Promise.reject(new Error('Could not encode image'))
  ctx.putImageData(data, 0, 0)
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (b) => {
        canvas.width = 0
        canvas.height = 0
        if (b) resolve(b)
        else reject(new Error('Could not encode image'))
      },
      'image/jpeg',
      quality
    )
  })
}

async function blobToImageData(blob: Blob): Promise<ImageData> {
  const bitmap = await createImageBitmap(blob)
  try {
    const canvas = document.createElement('canvas')
    canvas.width = bitmap.width
    canvas.height = bitmap.height
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('Could not read image')
    ctx.drawImage(bitmap, 0, 0)
    const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height)
    canvas.width = 0
    canvas.height = 0
    return imageData
  } finally {
    bitmap.close()
  }
}

export type PreparePageOptions = {
  mode?: ColourMode
  /** Auto-deskew — default false (quality regression when it false-positives). */
  deskew?: boolean
  quality?: number
}

export async function preparePageBlob(blob: Blob, opts: PreparePageOptions = {}): Promise<Blob> {
  const mode = opts.mode ?? 'original'
  const doDeskew = opts.deskew === true
  const quality = opts.quality ?? COLOUR_JPEG_QUALITY

  if (mode === 'original' && !doDeskew) return blob

  let imageData = await blobToImageData(blob)
  let changed = false
  if (doDeskew) {
    const d = deskewImageData(imageData)
    imageData = d.imageData
    if (d.applied) changed = true
  }
  if (mode !== 'original') {
    applyColourModeToImageData(imageData, mode)
    changed = true
  }
  if (!changed) return blob
  return imageDataToJpegBlob(imageData, quality)
}

export async function applyColourModeToBlob(blob: Blob, mode: ColourMode): Promise<Blob> {
  return preparePageBlob(blob, { mode, deskew: false })
}

/** Downscaled preview blob URL (caller must revoke). */
export async function previewColourModeUrl(
  blob: Blob,
  mode: ColourMode,
  maxEdge = 720
): Promise<string> {
  const bitmap = await createImageBitmap(blob)
  try {
    const scale = Math.min(1, maxEdge / Math.max(bitmap.width, bitmap.height))
    const w = Math.max(1, Math.round(bitmap.width * scale))
    const h = Math.max(1, Math.round(bitmap.height * scale))
    const canvas = document.createElement('canvas')
    canvas.width = w
    canvas.height = h
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('Could not preview colour mode')
    ctx.drawImage(bitmap, 0, 0, w, h)
    if (mode !== 'original') {
      const imageData = ctx.getImageData(0, 0, w, h)
      applyColourModeToImageData(imageData, mode)
      ctx.putImageData(imageData, 0, 0)
    }
    const out = await new Promise<Blob>((resolve, reject) => {
      canvas.toBlob(
        (b) => (b ? resolve(b) : reject(new Error('Could not encode preview'))),
        'image/jpeg',
        0.92
      )
    })
    canvas.width = 0
    canvas.height = 0
    return URL.createObjectURL(out)
  } finally {
    bitmap.close()
  }
}
