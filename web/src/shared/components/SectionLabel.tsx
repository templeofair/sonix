import type { ReactNode } from 'react'

type Props = {
  children: ReactNode
  className?: string
  as?: 'p' | 'span' | 'label' | 'h2'
}

const sectionLabelClass = 'text-xs font-semibold text-muted uppercase tracking-wider'

/** Uppercase section / field label used across AiPanel, Settings, review. */
export default function SectionLabel({ as: Tag = 'p', className = '', children }: Props) {
  return <Tag className={`${sectionLabelClass}${className ? ` ${className}` : ''}`}>{children}</Tag>
}

export { sectionLabelClass }
