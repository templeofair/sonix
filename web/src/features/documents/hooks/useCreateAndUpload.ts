import { useState } from 'react'
import { documentsApi } from '../services/documentsApi'
import type { UploadProgress } from '../lib/captureDraft'

/** Create a document and upload page files one-by-one (per-page progress). */
export function useCreateAndUpload() {
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState<UploadProgress | null>(null)

  const createAndUpload = async (
    title: string | undefined,
    files: File[],
    opts?: { onProgress?: (p: UploadProgress) => void }
  ) => {
    setUploading(true)
    setProgress({ current: 0, total: files.length })
    try {
      const { id } = await documentsApi.create(title)
      for (let i = 0; i < files.length; i++) {
        const p = { current: i, total: files.length }
        setProgress(p)
        opts?.onProgress?.(p)
        try {
          await documentsApi.uploadPages(id, [files[i]])
        } catch (err) {
          const msg = err instanceof Error ? err.message : 'Upload failed'
          throw new Error(`Page ${i + 1} of ${files.length} failed: ${msg}`)
        }
      }
      const done = { current: files.length, total: files.length }
      setProgress(done)
      opts?.onProgress?.(done)
      return id
    } finally {
      setUploading(false)
    }
  }

  return { uploading, progress, createAndUpload }
}
