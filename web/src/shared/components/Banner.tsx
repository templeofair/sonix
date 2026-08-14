import type { ReactNode } from 'react'

type Tone = 'error' | 'success' | 'warning'

const toneClass: Record<Tone, string> = {
  error: 'text-sm text-danger bg-danger-soft border border-danger-border/80 rounded-btn px-3 py-2',
  success: 'text-sm text-success bg-success-soft border border-success-border/80 rounded-btn px-3 py-2',
  warning: 'text-sm text-warning bg-warning-soft border border-warning-border/80 rounded-btn px-3 py-2',
}

type Props = {
  tone: Tone
  children: ReactNode
  className?: string
  /** Defaults to alert for error/warning, status for success. */
  role?: 'alert' | 'status'
}

/** Inline feedback banner — semantic success / warning / danger tokens. */
export default function Banner({ tone, children, className = '', role }: Props) {
  const resolvedRole = role ?? (tone === 'success' ? 'status' : 'alert')
  return (
    <p className={`${toneClass[tone]}${className ? ` ${className}` : ''}`} role={resolvedRole}>
      {children}
    </p>
  )
}
