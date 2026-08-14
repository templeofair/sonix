/** One page in the pre-save capture / upload review draft. */
export type DraftPage = {
  id: string
  blob: Blob
  url: string
  /** Clockwise degrees; baked into the JPEG on export. */
  rotation: 0 | 90 | 180 | 270
  source: 'camera' | 'upload'
  name?: string
}

export type UploadProgress = {
  /** 0-based index of the page currently uploading, or `total` when finished. */
  current: number
  total: number
}

let draftIdSeq = 0

export function newDraftPageId(): string {
  draftIdSeq += 1
  return `draft-${Date.now()}-${draftIdSeq}`
}

export function revokePageUrl(page: Pick<DraftPage, 'url'>): void {
  try {
    URL.revokeObjectURL(page.url)
  } catch {
    /* ignore */
  }
}

/** Apply clockwise rotation and return a JPEG blob (passthrough when rotation is 0). */
export async function blobWithRotation(blob: Blob, rotation: number): Promise<Blob> {
  const rot = ((rotation % 360) + 360) % 360
  if (rot === 0) return blob

  const bitmap = await createImageBitmap(blob)
  try {
    const canvas = document.createElement('canvas')
    if (rot === 90 || rot === 270) {
      canvas.width = bitmap.height
      canvas.height = bitmap.width
    } else {
      canvas.width = bitmap.width
      canvas.height = bitmap.height
    }
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('Could not rotate page')
    ctx.translate(canvas.width / 2, canvas.height / 2)
    ctx.rotate((rot * Math.PI) / 180)
    ctx.drawImage(bitmap, -bitmap.width / 2, -bitmap.height / 2)
    const out = await new Promise<Blob>((resolve, reject) => {
      canvas.toBlob(
        (b) => (b ? resolve(b) : reject(new Error('Could not encode rotated page'))),
        'image/jpeg',
        0.92
      )
    })
    return out
  } finally {
    bitmap.close()
  }
}

export async function draftPagesToFiles(pages: DraftPage[]): Promise<File[]> {
  return Promise.all(
    pages.map(async (p, i) => {
      const blob = await blobWithRotation(p.blob, p.rotation)
      const name = p.name?.replace(/\.[^.]+$/, '') || `page_${i}`
      return new File([blob], `${name}.jpg`, { type: 'image/jpeg' })
    })
  )
}

export function moveIndex(from: number, to: number, length: number): number {
  if (length <= 0) return 0
  if (to < 0) return 0
  if (to >= length) return length - 1
  if (from < 0 || from >= length) return from
  return to
}

export function reorderList<T>(list: T[], from: number, to: number): T[] {
  if (from === to || from < 0 || to < 0 || from >= list.length || to >= list.length) return list
  const next = list.slice()
  const [item] = next.splice(from, 1)
  next.splice(to, 0, item)
  return next
}
