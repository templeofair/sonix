import type { ButtonHTMLAttributes, ReactNode } from 'react'

type Variant = 'primary' | 'secondary' | 'danger'

const variantClass: Record<Variant, string> = {
  primary:
    'control bg-accent text-white rounded-btn font-medium shadow-sm hover:opacity-95 disabled:opacity-50 transition-opacity',
  secondary:
    'control border border-border rounded-btn font-medium text-gray-800 bg-white hover:bg-surface disabled:opacity-50 transition-colors',
  danger:
    'control rounded-btn font-medium bg-danger text-white shadow-sm hover:opacity-90 disabled:opacity-50',
}

type Props = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: Variant
  children: ReactNode
}

/** Shared button — class strings match existing primary/secondary/danger controls.
 *  Also mounted from the DEV `/__ui` primitives gallery (`web/src/mocks/ui`). */
export default function Button({
  variant = 'primary',
  className = '',
  type = 'button',
  children,
  ...rest
}: Props) {
  return (
    <button type={type} className={`${variantClass[variant]}${className ? ` ${className}` : ''}`} {...rest}>
      {children}
    </button>
  )
}

/** For Link / non-button surfaces that should look like a Button. */
export function buttonVariantClass(variant: Variant = 'primary'): string {
  return variantClass[variant]
}
