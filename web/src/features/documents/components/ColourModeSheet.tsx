import { useEffect, useRef, useState } from 'react'
import {
  applyColourModeToBlob,
  loadColourPrefs,
  previewColourModeUrl,
  saveColourPrefs,
  type ColourMode,
} from '../lib/colourMode'
import { blobWithRotation } from '../lib/captureDraft'
import CaptureEditorShell, { captureShellSecondaryBtn } from './CaptureEditorShell'
import ColourModeControls from './ColourModeControls'

type PageLike = {
  blob: Blob
  rotation: number
  source: 'camera' | 'upload'
}

type Props = {
  page: PageLike
  pageCount: number
  onCancel: () => void
  onApply: (blob: Blob) => void
  onApplyAll: (blobs: Blob[]) => void
  allPages: PageLike[]
}

export default function ColourModeSheet({
  page,
  pageCount,
  onCancel,
  onApply,
  onApplyAll,
  allPages,
}: Props) {
  const prefs = loadColourPrefs()
  const [mode, setMode] = useState<ColourMode>(prefs.mode)
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const previewRef = useRef<string | null>(null)

  useEffect(() => {
    saveColourPrefs(mode)
  }, [mode])

  useEffect(() => {
    let cancelled = false
    const run = async () => {
      try {
        const oriented = await blobWithRotation(page.blob, page.rotation)
        const url = await previewColourModeUrl(oriented, mode)
        if (cancelled) {
          URL.revokeObjectURL(url)
          return
        }
        if (previewRef.current) URL.revokeObjectURL(previewRef.current)
        previewRef.current = url
        setPreviewUrl(url)
      } catch {
        if (!cancelled) setError('Could not preview.')
      }
    }
    void run()
    return () => {
      cancelled = true
    }
  }, [page.blob, page.rotation, mode])

  useEffect(() => {
    return () => {
      if (previewRef.current) URL.revokeObjectURL(previewRef.current)
    }
  }, [])

  const applyOne = async () => {
    setBusy(true)
    setError(null)
    try {
      const oriented = await blobWithRotation(page.blob, page.rotation)
      const out = await applyColourModeToBlob(oriented, mode)
      onApply(out)
    } catch {
      setError('Could not apply colour mode.')
      setBusy(false)
    }
  }

  const applyAll = async () => {
    setBusy(true)
    setError(null)
    try {
      const blobs: Blob[] = []
      for (const p of allPages) {
        const oriented = await blobWithRotation(p.blob, p.rotation)
        blobs.push(await applyColourModeToBlob(oriented, mode))
      }
      onApplyAll(blobs)
    } catch {
      setError('Could not apply to all pages.')
      setBusy(false)
    }
  }

  return (
    <CaptureEditorShell
      title="Colour"
      ariaLabel="Colour mode"
      onCancel={onCancel}
      onDone={() => void applyOne()}
      doneDisabled={!previewUrl}
      doneBusy={busy}
      footer={
        <div className="space-y-3">
          <ColourModeControls mode={mode} onModeChange={setMode} variant="dark" />
          {pageCount > 1 ? (
            <button
              type="button"
              onClick={() => void applyAll()}
              disabled={busy}
              className={`${captureShellSecondaryBtn} w-full`}
            >
              {busy ? 'Applying…' : 'Apply to all pages'}
            </button>
          ) : null}
        </div>
      }
    >
      <div className="relative flex-1 min-h-0 flex items-center justify-center p-4">
        {previewUrl ? (
          <img
            src={previewUrl}
            alt="Colour preview"
            className="max-h-full max-w-full object-contain select-none"
          />
        ) : (
          <p className="text-white/60 text-sm">Preparing preview…</p>
        )}
        {error ? (
          <p
            className="absolute bottom-4 left-4 right-4 text-sm text-red-200 bg-red-950/80 border border-red-500/40 rounded-btn px-3 py-2"
            role="alert"
          >
            {error}
          </p>
        ) : null}
      </div>
    </CaptureEditorShell>
  )
}
