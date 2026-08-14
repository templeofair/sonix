import { useCallback } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import type { ReactNode } from 'react'

const baseClass =
  'control inline-flex items-center justify-center px-4 py-2 rounded-btn border border-border bg-white text-sm font-medium text-gray-800 shadow-sm hover:bg-surface hover:border-accent/25 focus-visible:ring-offset-2 focus-visible:ring-offset-card transition-colors'

type LinkProps = { to: string; children: ReactNode; className?: string }

type ButtonProps = { onClick: () => void; children: ReactNode; className?: string }

/** Harmonized “up” control — no arrow glyphs; reads as a secondary button. */
export function NavBackLink({ to, children, className = '' }: LinkProps) {
  return (
    <Link to={to} className={`${baseClass} ${className}`.trim()}>
      {children}
    </Link>
  )
}

/** Same styling as NavBackLink for history navigation or custom handlers. */
export function NavBackButton({ onClick, children, className = '' }: ButtonProps) {
  return (
    <button type="button" onClick={onClick} className={`${baseClass} ${className}`.trim()}>
      {children}
    </button>
  )
}

type HistoryBackProps = { fallbackTo: string; children?: ReactNode; className?: string; 'aria-label'?: string }

/** “Back” — `navigate(-1)` when history allows; otherwise `navigate(fallbackTo)`. */
export function NavBackHistoryButton({
  fallbackTo,
  children = 'Back',
  className = '',
  'aria-label': ariaLabel,
}: HistoryBackProps) {
  const navigate = useNavigate()
  const onClick = useCallback(() => {
    if (typeof window !== 'undefined' && window.history.length > 1) {
      navigate(-1)
    } else {
      navigate(fallbackTo)
    }
  }, [navigate, fallbackTo])
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={ariaLabel}
      className={`${baseClass} ${className}`.trim()}
    >
      {children}
    </button>
  )
}
