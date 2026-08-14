import { useEffect, type ReactNode } from 'react'

export const captureShellSecondaryBtn =
  'control min-h-[44px] px-3 rounded-btn text-sm font-medium bg-white/10 hover:bg-white/20 focus:outline-none focus-visible:ring-2 focus-visible:ring-white/40 disabled:opacity-50'

export const captureShellPrimaryBtn =
  'control min-h-[44px] px-3 rounded-btn text-sm font-medium bg-accent text-white disabled:opacity-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/50'

type Props = {
  title: string
  onCancel: () => void
  cancelLabel?: string
  onDone?: () => void
  doneLabel?: string
  doneDisabled?: boolean
  doneBusy?: boolean
  children: ReactNode
  footer?: ReactNode
  /** Replaces centred title (e.g. document name field on camera). */
  center?: ReactNode
  zClass?: string
  /** Accessible name when title is replaced by `center`. */
  ariaLabel?: string
}

/** Full-screen black editor chrome shared by crop, colour, and camera. */
export default function CaptureEditorShell({
  title,
  onCancel,
  cancelLabel = 'Cancel',
  onDone,
  doneLabel = 'Done',
  doneDisabled,
  doneBusy,
  children,
  footer,
  center,
  zClass = 'z-[110]',
  ariaLabel,
}: Props) {
  useEffect(() => {
    const onKey = (e: globalThis.KeyboardEvent) => {
      if (e.key === 'Escape') onCancel()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onCancel])

  return (
    <div
      className={`fixed inset-0 ${zClass} flex h-[100dvh] max-h-[100dvh] flex-col overflow-hidden bg-black text-white`}
      role="dialog"
      aria-modal="true"
      aria-label={ariaLabel ?? title}
    >
      <header className="flex-shrink-0 flex items-center justify-between gap-2 px-3 py-2 border-b border-white/10 bg-gray-950/95">
        <button type="button" onClick={onCancel} className={captureShellSecondaryBtn}>
          {cancelLabel}
        </button>
        {center ? (
          <div className="flex-1 min-w-0">{center}</div>
        ) : (
          <h1 className="flex-1 text-center text-sm font-semibold truncate px-1">{title}</h1>
        )}
        {onDone ? (
          <button
            type="button"
            onClick={onDone}
            disabled={doneDisabled || doneBusy}
            className={captureShellPrimaryBtn}
          >
            {doneBusy ? 'Applying…' : doneLabel}
          </button>
        ) : (
          <span className="min-w-[4.5rem]" aria-hidden />
        )}
      </header>
      <div className="flex-1 min-h-0 flex flex-col">{children}</div>
      {footer ? (
        <div className="flex-shrink-0 border-t border-white/10 bg-gray-950/95 px-3 py-3 pb-[calc(0.75rem+env(safe-area-inset-bottom,0px))]">
          {footer}
        </div>
      ) : null}
    </div>
  )
}
