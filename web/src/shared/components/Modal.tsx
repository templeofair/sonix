import { useEffect, useRef, type MouseEvent, type ReactNode } from 'react'
import { createPortal } from 'react-dom'

const FOCUSABLE = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(', ')

type Props = {
  /** Escape and explicit close actions route here. Backdrop may also call this when dismiss ≠ none. */
  onClose: () => void
  /** id of the heading that names the dialog. */
  labelledBy: string
  /** Optional id of supporting text (aria-describedby). */
  describedBy?: string
  /** Panel box — size and inner layout differ per dialog. */
  panelClassName?: string
  /** Stacking layer; the capture flow sits above the app shell. */
  overlayClassName?: string
  /**
   * Backdrop dismiss: `click` / `mousedown` when the event target is the overlay;
   * `none` keeps the overlay (e.g. rename confirm that must not dismiss on backdrop).
   */
  dismiss?: 'click' | 'mousedown' | 'none'
  /** Fires on any mousedown that reaches the overlay (including from the panel). */
  onOverlayMouseDown?: () => void
  children: ReactNode
}

/**
 * Centred dialog with the shared backdrop, a focus trap, Escape to close, and
 * focus returned to the opener on unmount. Mounted on `document.body` so nested
 * z-index / overflow stacking contexts (e.g. library cards) cannot cover it.
 * Full-screen editor chrome uses CaptureEditorShell instead.
 */
export default function Modal({
  onClose,
  labelledBy,
  describedBy,
  panelClassName = '',
  overlayClassName = 'z-50 p-4',
  dismiss = 'click',
  onOverlayMouseDown,
  children,
}: Props) {
  const panelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const opener = document.activeElement as HTMLElement | null
    panelRef.current?.focus()
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prevOverflow
      opener?.focus?.()
    }
  }, [])

  useEffect(() => {
    const onKey = (e: globalThis.KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
        return
      }
      if (e.key !== 'Tab') return
      const panel = panelRef.current
      if (!panel) return
      const items = Array.from(panel.querySelectorAll<HTMLElement>(FOCUSABLE))
      if (items.length === 0) {
        e.preventDefault()
        panel.focus()
        return
      }
      const first = items[0]
      const last = items[items.length - 1]
      const active = document.activeElement
      const outside = !active || !panel.contains(active)
      if (!e.shiftKey && (active === last || outside)) {
        e.preventDefault()
        first.focus()
      } else if (e.shiftKey && (active === first || active === panel || outside)) {
        e.preventDefault()
        last.focus()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  return createPortal(
    <div
      className={`fixed inset-0 flex items-center justify-center bg-black/45 backdrop-blur-sm ${overlayClassName}`}
      role="presentation"
      onClick={
        dismiss === 'click'
          ? (e: MouseEvent<HTMLDivElement>) => {
              if (e.target === e.currentTarget) onClose()
            }
          : undefined
      }
      onMouseDown={(e: MouseEvent<HTMLDivElement>) => {
        onOverlayMouseDown?.()
        if (dismiss === 'mousedown' && e.target === e.currentTarget) onClose()
      }}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={labelledBy}
        aria-describedby={describedBy}
        tabIndex={-1}
        className={`bg-card border border-border rounded-card shadow-2xl focus:outline-none ${panelClassName}`}
      >
        {children}
      </div>
    </div>,
    document.body
  )
}
