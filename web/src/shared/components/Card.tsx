import type { HTMLAttributes, ReactNode } from 'react'

type Props = HTMLAttributes<HTMLElement> & {
  children: ReactNode
  as?: 'div' | 'section'
}

const cardClass = 'rounded-card border border-border bg-card shadow-card'

/** Shared card shell — same classes used across Settings, Login, library. */
export default function Card({ as: Tag = 'div', className = '', children, ...rest }: Props) {
  return (
    <Tag className={`${cardClass}${className ? ` ${className}` : ''}`} {...rest}>
      {children}
    </Tag>
  )
}

export { cardClass }
