import type { ReactNode } from 'react'

type Props = {
  title: string
  children?: ReactNode
  action?: ReactNode
  className?: string
}

/** Dashed empty / idle card used by library and load placeholders. */
export default function EmptyState({ title, children, action, className = '' }: Props) {
  return (
    <div
      className={`rounded-card border border-dashed border-border bg-card/50 px-6 py-10 text-center space-y-3${className ? ` ${className}` : ''}`}
    >
      <p className="text-sm text-muted">{title}</p>
      {children}
      {action}
    </div>
  )
}
