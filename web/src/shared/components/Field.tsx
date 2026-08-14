import type { InputHTMLAttributes, ReactNode } from 'react'

/** Shared text-field input classes (focus-visible; 16px mobile via index.css base). */
export const fieldInputClass =
  'w-full px-3 py-2.5 border border-border rounded-btn text-base md:text-sm text-gray-900 bg-white placeholder:text-muted-subtle focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/30 focus-visible:border-accent'

type Props = {
  id: string
  label: string
  children?: ReactNode
  /** Optional control inside the input (e.g. show/hide password). */
  endAdornment?: ReactNode
} & InputHTMLAttributes<HTMLInputElement>

/** Label + input used by Login and Settings. */
export default function Field({ id, label, className = '', children, endAdornment, ...rest }: Props) {
  const inputClass = `${fieldInputClass}${endAdornment ? ' pr-12' : ''}${className ? ` ${className}` : ''}`
  return (
    <div>
      <label htmlFor={id} className="block text-xs font-semibold text-muted uppercase tracking-wider mb-2">
        {label}
      </label>
      {endAdornment ? (
        <div className="relative">
          <input id={id} className={inputClass} {...rest} />
          <div className="absolute inset-y-0 right-0 flex items-center pr-1">{endAdornment}</div>
        </div>
      ) : (
        <input id={id} className={inputClass} {...rest} />
      )}
      {children}
    </div>
  )
}
