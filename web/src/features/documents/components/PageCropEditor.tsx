import { useCallback, useEffect, useRef, useState } from 'react'
import type { Point, Quad } from '../lib/perspectiveCrop'
import {
  clampPoint,
  defaultQuad,
  fullPageQuad,
  perspectiveCropBlob,
} from '../lib/perspectiveCrop'
import CaptureEditorShell, { captureShellSecondaryBtn } from './CaptureEditorShell'

type Props = {
  imageUrl: string
  onCancel: () => void
  onConfirm: (blob: Blob) => void
}

const HANDLE = 44
const LOUPE = 96
const LOUPE_ZOOM = 1.5

type DragState = { corner: number; pointerId: number }

export default function PageCropEditor({ imageUrl, onCancel, onConfirm }: Props) {
  const imgRef = useRef<HTMLImageElement>(null)
  const stageRef = useRef<HTMLDivElement>(null)
  const [nat, setNat] = useState({ w: 0, h: 0 })
  const [quad, setQuad] = useState<Quad | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [drag, setDrag] = useState<DragState | null>(null)
  const [loupe, setLoupe] = useState<{ x: number; y: number; ix: number; iy: number } | null>(null)
  const initialQuad = useRef<Quad | null>(null)

  useEffect(() => {
    const img = new Image()
    img.onload = () => {
      const w = img.naturalWidth
      const h = img.naturalHeight
      setNat({ w, h })
      const q = defaultQuad(w, h)
      setQuad(q)
      initialQuad.current = q
    }
    img.onerror = () => setError('Could not load page image.')
    img.src = imageUrl
  }, [imageUrl])

  const displayToImage = useCallback(
    (clientX: number, clientY: number): Point | null => {
      const el = imgRef.current
      if (!el || !nat.w) return null
      const r = el.getBoundingClientRect()
      if (r.width < 1 || r.height < 1) return null
      const x = ((clientX - r.left) / r.width) * nat.w
      const y = ((clientY - r.top) / r.height) * nat.h
      return clampPoint({ x, y }, nat.w, nat.h)
    },
    [nat]
  )

  const imageToDisplay = useCallback(
    (p: Point): { left: number; top: number } | null => {
      const el = imgRef.current
      if (!el || !nat.w) return null
      const r = el.getBoundingClientRect()
      const stage = stageRef.current?.getBoundingClientRect()
      if (!stage) return null
      return {
        left: r.left - stage.left + (p.x / nat.w) * r.width,
        top: r.top - stage.top + (p.y / nat.h) * r.height,
      }
    },
    [nat]
  )

  const onPointerDown = (corner: number) => (e: React.PointerEvent) => {
    e.preventDefault()
    e.stopPropagation()
    ;(e.target as HTMLElement).setPointerCapture(e.pointerId)
    setDrag({ corner, pointerId: e.pointerId })
    const p = displayToImage(e.clientX, e.clientY)
    if (p) {
      setLoupe({ x: e.clientX, y: e.clientY, ix: p.x, iy: p.y })
    }
  }

  const onPointerMove = (e: React.PointerEvent) => {
    if (!drag || e.pointerId !== drag.pointerId || !quad) return
    const p = displayToImage(e.clientX, e.clientY)
    if (!p) return
    setQuad((prev) => {
      if (!prev) return prev
      const next = [...prev] as Quad
      next[drag.corner] = p
      return next
    })
    setLoupe({ x: e.clientX, y: e.clientY - LOUPE * 0.85, ix: p.x, iy: p.y })
  }

  const onPointerUp = (e: React.PointerEvent) => {
    if (!drag || e.pointerId !== drag.pointerId) return
    setDrag(null)
    setLoupe(null)
  }

  const reset = () => {
    if (!nat.w) return
    const q = defaultQuad(nat.w, nat.h)
    setQuad(q)
    initialQuad.current = q
  }

  const useFullPage = () => {
    if (!nat.w) return
    setQuad(fullPageQuad(nat.w, nat.h))
  }

  const confirm = async () => {
    if (!quad || busy) return
    setBusy(true)
    setError(null)
    try {
      const res = await fetch(imageUrl)
      const srcBlob = await res.blob()
      const { blob } = await perspectiveCropBlob(srcBlob, quad)
      onConfirm(blob)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Crop failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <CaptureEditorShell
      title="Crop page"
      onCancel={onCancel}
      onDone={() => void confirm()}
      doneDisabled={!quad}
      doneBusy={busy}
      footer={
        <div className="flex flex-wrap gap-2">
          <button type="button" onClick={reset} className={`${captureShellSecondaryBtn} flex-1`}>
            Reset
          </button>
          <button type="button" onClick={useFullPage} className={`${captureShellSecondaryBtn} flex-1`}>
            Full page
          </button>
        </div>
      }
    >
      <div
        ref={stageRef}
        className="relative flex-1 min-h-0 overflow-hidden touch-none"
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onPointerCancel={onPointerUp}
      >
        <img
          ref={imgRef}
          src={imageUrl}
          alt="Page to crop"
          draggable={false}
          className="absolute inset-0 m-auto max-w-full max-h-full object-contain select-none pointer-events-none"
        />

        {quad && nat.w > 0
          ? (() => {
              const pts = quad.map(imageToDisplay)
              if (pts.some((p) => !p)) return null
              const poly = pts.map((p) => `${p!.left},${p!.top}`).join(' ')
              return (
                <>
                  <svg className="absolute inset-0 w-full h-full pointer-events-none" aria-hidden>
                    <polygon
                      points={poly}
                      fill="rgba(37, 99, 235, 0.15)"
                      stroke="rgb(96, 165, 250)"
                      strokeWidth="2"
                    />
                  </svg>
                  {quad.map((_, i) => {
                    const pos = pts[i]!
                    return (
                      <button
                        key={i}
                        type="button"
                        aria-label={`Corner ${i + 1}`}
                        onPointerDown={onPointerDown(i)}
                        className="absolute z-10 rounded-full bg-white border-2 border-accent shadow-md touch-none"
                        style={{
                          width: HANDLE,
                          height: HANDLE,
                          left: pos.left - HANDLE / 2,
                          top: pos.top - HANDLE / 2,
                        }}
                      >
                        <span className="block w-3 h-3 mx-auto rounded-full bg-accent" />
                      </button>
                    )
                  })}
                </>
              )
            })()
          : null}

        {loupe && nat.w > 0 ? (
          <div
            className="pointer-events-none absolute z-20 rounded-full border-2 border-white shadow-lg overflow-hidden bg-black"
            style={{
              width: LOUPE,
              height: LOUPE,
              left: Math.min(
                (stageRef.current?.clientWidth ?? LOUPE) - LOUPE - 8,
                Math.max(8, loupe.x - (stageRef.current?.getBoundingClientRect().left ?? 0) - LOUPE / 2)
              ),
              top: Math.max(
                8,
                loupe.y - (stageRef.current?.getBoundingClientRect().top ?? 0) - LOUPE / 2
              ),
            }}
            aria-hidden
          >
            <div
              className="w-full h-full"
              style={{
                backgroundImage: `url(${imageUrl})`,
                backgroundRepeat: 'no-repeat',
                backgroundSize: `${nat.w * LOUPE_ZOOM}px ${nat.h * LOUPE_ZOOM}px`,
                backgroundPosition: `${LOUPE / 2 - loupe.ix * LOUPE_ZOOM}px ${LOUPE / 2 - loupe.iy * LOUPE_ZOOM}px`,
              }}
            />
          </div>
        ) : null}

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
