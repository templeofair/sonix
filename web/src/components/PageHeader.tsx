import type { ReactNode } from 'react'
import { useAuth } from '../auth'

type Props = {
  /** Ignored when `titleSlot` is set */
  title?: string
  /** Custom title row (e.g. editable document title + status) — takes precedence over `title` */
  titleSlot?: ReactNode
  /** Shown under the default `h1` when `titleSlot` is not used */
  subtitle?: ReactNode
  left?: ReactNode
  right?: ReactNode
  /** When false, hide default desktop Log out in the header (rare). Default true. */
  showLogout?: boolean
}

/**
 * Sticky-style page title bar: title left, optional back control, desktop Log out + optional actions right.
 * Fixed h-16 so the bottom border lines up with the sidebar Sonix brand bar.
 */
export default function PageHeader({ title, titleSlot, left, right, subtitle, showLogout = true }: Props) {
  const { logout } = useAuth()

  const logoutBtn = showLogout ? (
    <button
      type="button"
      onClick={() => void logout()}
      className="hidden md:inline-flex items-center px-3 py-2 text-sm font-medium text-muted hover:text-gray-900 rounded-btn hover:bg-surface transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/40 focus-visible:ring-offset-2 focus-visible:ring-offset-card"
    >
      Log out
    </button>
  ) : null

  const rightSlot =
    logoutBtn != null || right != null ? (
      <div className="flex items-center flex-shrink-0 gap-2">
        {/* Page actions first; Log out last (far right). */}
        {right}
        {logoutBtn}
      </div>
    ) : null

  return (
    <header className="h-16 border-b border-border bg-card px-4 sm:px-6 flex items-center gap-3 flex-shrink-0">
      {left != null ? <div className="flex items-center flex-shrink-0">{left}</div> : null}
      <div className="min-w-0 flex-1">
        {titleSlot != null ? (
          titleSlot
        ) : (
          <>
            {title != null ? (
              <h1 className="text-lg font-semibold text-gray-900 tracking-tight truncate leading-tight">{title}</h1>
            ) : null}
            {subtitle != null ? <div className="text-xs text-muted mt-0.5 truncate leading-tight">{subtitle}</div> : null}
          </>
        )}
      </div>
      {rightSlot}
    </header>
  )
}
