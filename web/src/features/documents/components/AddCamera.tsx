import { useCallback, useEffect, useRef } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useCameraCapture } from '../hooks/useCameraCapture'
import { useCaptureDraft } from '../hooks/CaptureDraftContext'
import CaptureEditorShell from './CaptureEditorShell'
import { useAppNav } from '../../../lib/appNav'

/**
 * Capture-only camera. Colour / crop / rotate happen on `/add/review`
 * after the shot. Cancel returns to review when opened from there (draft kept).
 */
export default function AddCamera() {
  const navigate = useNavigate()
  const { appPath } = useAppNav()
  const location = useLocation()
  const resumeDraft = Boolean(
    (location.state as { resumeDraft?: boolean } | null)?.resumeDraft
  )
  const draft = useCaptureDraft()
  const clearedRef = useRef(false)
  const {
    streaming,
    videoReady,
    cameraError,
    flash,
    torchSupported,
    torchOn,
    focusSupported,
    videoRef,
    streamRef,
    startCamera,
    stopCamera,
    capture,
    setTorch,
    tapToFocus,
  } = useCameraCapture()

  useEffect(() => {
    if (!resumeDraft && !clearedRef.current) {
      clearedRef.current = true
      draft.clear()
    }
  }, [resumeDraft, draft])

  const returnToReview = useCallback(() => {
    stopCamera()
    draft.setRetakeIndex(null)
    navigate(appPath('/add/review'))
  }, [stopCamera, draft, navigate, appPath])

  const abandonCapture = useCallback(() => {
    stopCamera()
    draft.clear()
    navigate(appPath('/add'))
  }, [stopCamera, draft, navigate, appPath])

  /** Cancel keeps the draft when returning from review; abandons on a fresh scan. */
  const onCancel = useCallback(() => {
    if (resumeDraft && draft.pages.length > 0) {
      returnToReview()
    } else {
      abandonCapture()
    }
  }, [resumeDraft, draft.pages.length, returnToReview, abandonCapture])

  useEffect(() => {
    startCamera()
    return () => {
      streamRef.current?.getTracks().forEach((t) => t.stop())
      streamRef.current = null
    }
  }, [startCamera, streamRef])

  const onCapture = useCallback(async () => {
    const blob = await capture()
    if (!blob) return
    if (draft.retakeIndex != null) {
      draft.replaceAt(draft.retakeIndex, blob, 'camera')
    } else {
      draft.addBlob(blob, 'camera')
    }
  }, [capture, draft])

  const goReview = useCallback(() => {
    if (draft.pages.length === 0) return
    returnToReview()
  }, [draft.pages.length, returnToReview])

  const onViewfinderPointer = useCallback(
    (e: React.PointerEvent<HTMLDivElement>) => {
      if (!focusSupported || e.button !== 0) return
      const rect = e.currentTarget.getBoundingClientRect()
      if (rect.width <= 0 || rect.height <= 0) return
      const nx = (e.clientX - rect.left) / rect.width
      const ny = (e.clientY - rect.top) / rect.height
      void tapToFocus(nx, ny)
    },
    [focusSupported, tapToFocus]
  )

  const titleField = (
    <input
      type="text"
      value={draft.title}
      onChange={(e) => draft.setTitle(e.target.value)}
      placeholder="Document name…"
      className="w-full bg-white/10 text-white text-sm px-2 py-1.5 rounded-btn placeholder-white/40 border border-white/20 focus:outline-none focus:ring-2 focus:ring-accent/40 focus:border-white/40"
    />
  )

  return (
    <CaptureEditorShell
      title="Camera"
      ariaLabel="Camera"
      zClass="z-[100]"
      onCancel={onCancel}
      onDone={goReview}
      doneDisabled={draft.pages.length === 0}
      center={titleField}
    >
      <div
        className="relative flex-1 min-h-0 overflow-hidden overscroll-none touch-none"
        onPointerUp={onViewfinderPointer}
      >
        {cameraError && !streaming && (
          <div className="absolute inset-0 flex items-center justify-center z-10 p-6">
            <div className="bg-gray-900/90 rounded-card border border-white/15 p-6 max-w-sm w-full space-y-4 text-center">
              <p className="text-white/90 text-sm leading-relaxed">{cameraError}</p>
              {typeof window !== 'undefined' &&
                window.location.protocol !== 'https:' &&
                window.location.hostname !== 'localhost' &&
                window.location.hostname !== '127.0.0.1' && (
                  <a
                    href={`https://${window.location.hostname}:9443${window.location.pathname}`}
                    className="inline-block px-5 py-2.5 bg-accent text-white rounded-btn font-medium text-sm shadow-sm"
                  >
                    Open via HTTPS
                  </a>
                )}
              <button
                type="button"
                onClick={startCamera}
                className="block w-full text-accent font-medium text-sm py-2 rounded-btn hover:bg-white/10"
              >
                Try again
              </button>
            </div>
          </div>
        )}

        {!streaming && !cameraError && (
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="text-white/50 text-sm flex flex-col items-center gap-3">
              <svg className="animate-spin" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83" />
              </svg>
              Opening camera…
            </div>
          </div>
        )}

        <video
          ref={videoRef}
          autoPlay
          playsInline
          muted
          className={`w-full h-full object-cover ${streaming && videoReady ? 'opacity-100' : 'opacity-0'}`}
        />

        {flash && <div className="absolute inset-0 bg-white z-20 pointer-events-none" />}

        {streaming && videoReady && (
          <div className="absolute inset-0 pointer-events-none">
            <div className="absolute inset-6 sm:inset-10 border-2 border-white/30 rounded-xl" />
          </div>
        )}
      </div>

      {streaming && (
        <div className="flex-shrink-0 border-t border-white/10 bg-gray-950/95 pb-[env(safe-area-inset-bottom)]">
          {draft.pages.length > 0 && (
            <div className="flex gap-2 px-4 pt-3 pb-1 overflow-x-auto overflow-y-hidden flex-nowrap touch-manipulation">
              {draft.pages.map((page, i) => (
                <div
                  key={page.id}
                  className={`relative flex-shrink-0 w-12 h-16 rounded-btn overflow-hidden border-2 ${
                    draft.retakeIndex === i ? 'border-accent' : 'border-white/25'
                  }`}
                >
                  <img
                    src={page.url}
                    alt={`Page ${i + 1}`}
                    className="w-full h-full object-cover"
                    style={{ transform: `rotate(${page.rotation}deg)` }}
                  />
                  <button
                    type="button"
                    onClick={() => draft.removeAt(i)}
                    className="absolute -top-1 -right-1 min-h-[28px] min-w-[28px] bg-red-500 text-white text-sm flex items-center justify-center rounded-full shadow focus:outline-none focus-visible:ring-2 focus-visible:ring-white"
                    aria-label={`Remove page ${i + 1}`}
                  >
                    ×
                  </button>
                  <span className="absolute bottom-0 left-0 right-0 bg-black/60 text-white text-[10px] text-center py-px">
                    {i + 1}
                  </span>
                </div>
              ))}
            </div>
          )}

          <div className="relative flex items-center justify-between px-6 py-4 min-h-[5.5rem]">
            <div className="w-16 shrink-0 flex justify-start">
              {torchSupported ? (
                <button
                  type="button"
                  onClick={() => void setTorch(!torchOn)}
                  aria-pressed={torchOn}
                  aria-label={torchOn ? 'Turn torch off' : 'Turn torch on'}
                  className={`min-h-[44px] min-w-[44px] rounded-full border text-sm font-medium ${
                    torchOn
                      ? 'border-amber-300 bg-amber-400/90 text-gray-900'
                      : 'border-white/30 bg-white/10 text-white'
                  }`}
                >
                  Flash
                </button>
              ) : (
                <span className="w-11" aria-hidden />
              )}
            </div>

            <button
              type="button"
              onClick={() => void onCapture()}
              disabled={!videoReady}
              className="absolute left-1/2 -translate-x-1/2 bottom-4 w-[72px] h-[72px] rounded-full border-[4px] border-white flex items-center justify-center disabled:opacity-40 active:scale-90 transition-transform"
              aria-label="Take photo"
            >
              <span className="block w-[58px] h-[58px] rounded-full bg-white active:bg-gray-200 transition-colors" />
            </button>

            <span className="w-16 shrink-0 text-right text-xs text-white/50 tabular-nums">
              {draft.pages.length > 0 ? `${draft.pages.length} pg` : ''}
            </span>
          </div>
        </div>
      )}
    </CaptureEditorShell>
  )
}
