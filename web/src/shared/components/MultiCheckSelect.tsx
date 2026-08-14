import { useEffect, useId, useRef, useState } from 'react'

export type MultiCheckOption = {
  value: string
  label: string
}

type Props = {
  id?: string
  label: string
  options: MultiCheckOption[]
  values: string[]
  onChange: (next: string[]) => void
  /** Shown when nothing is selected (no filter). */
  emptyLabel?: string
  className?: string
}

/** Checkbox dropdown for 0–n filter values. Empty selection = no preference. */
export default function MultiCheckSelect({
  id: idProp,
  label,
  options,
  values,
  onChange,
  emptyLabel = 'Any',
  className = '',
}: Props) {
  const autoId = useId()
  const id = idProp ?? autoId
  const panelId = `${id}-panel`
  const rootRef = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)

  useEffect(() => {
    if (!open) return
    const onDoc = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDoc)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const selected = new Set(values)
  const summary =
    values.length === 0
      ? emptyLabel
      : values.length <= 2
        ? options
            .filter((o) => selected.has(o.value))
            .map((o) => o.label)
            .join(', ')
        : `${values.length} selected`

  const toggle = (value: string) => {
    if (selected.has(value)) onChange(values.filter((v) => v !== value))
    else onChange([...values, value])
    setOpen(false)
  }

  return (
    <div ref={rootRef} className={`relative min-w-0 ${className}`}>
      <label htmlFor={id} className="block text-xs font-semibold text-muted uppercase tracking-wider mb-1.5">
        {label}
      </label>
      <button
        id={id}
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={panelId}
        onClick={() => setOpen((v) => !v)}
        className="control w-full min-h-[44px] px-3 rounded-btn border border-border bg-white text-sm text-left text-gray-900 inline-flex items-center justify-between gap-2 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/30 focus-visible:border-accent"
      >
        <span className={`truncate ${values.length === 0 ? 'text-muted-subtle' : ''}`}>{summary}</span>
        <span aria-hidden className="text-muted flex-shrink-0">
          {open ? '▴' : '▾'}
        </span>
      </button>
      {open ? (
        <div
          id={panelId}
          role="listbox"
          aria-multiselectable
          aria-label={label}
          className="absolute z-30 mt-1 w-full max-h-56 overflow-y-auto rounded-btn border border-border bg-card shadow-card py-1"
        >
          {options.length === 0 ? (
            <p className="px-3 py-2 text-sm text-muted">Nothing to choose</p>
          ) : (
            options.map((o) => {
              const checked = selected.has(o.value)
              const optId = `${id}-opt-${o.value}`
              return (
                <label
                  key={o.value}
                  htmlFor={optId}
                  className="flex items-center gap-2 min-h-[44px] px-3 cursor-pointer hover:bg-surface text-sm text-gray-900"
                >
                  <input
                    id={optId}
                    type="checkbox"
                    className="h-4 w-4 accent-[var(--color-accent,theme(colors.accent.DEFAULT))]"
                    checked={checked}
                    onChange={() => toggle(o.value)}
                  />
                  <span className="truncate">{o.label}</span>
                </label>
              )
            })
          )}
          {values.length > 0 ? (
            <button
              type="button"
              className="control w-full min-h-[44px] px-3 text-left text-sm text-accent border-t border-border hover:bg-surface"
              onClick={() => {
                onChange([])
                setOpen(false)
              }}
            >
              Clear
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}
