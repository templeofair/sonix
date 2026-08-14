type Props = {
  className?: string
  label?: string
  /**
   * When true (default if `label` is set), expose role="status".
   * Set false when a parent already owns the live region.
   */
  status?: boolean
}

/** Inline spinner matching existing AiPanel / camera markup. */
export default function Spinner({ className = 'w-5 h-5', label, status }: Props) {
  const asStatus = status ?? Boolean(label)
  return (
    <span
      className="inline-flex items-center text-muted"
      role={asStatus ? 'status' : undefined}
      aria-hidden={asStatus ? undefined : true}
    >
      <svg className={`${className} animate-spin`} fill="none" viewBox="0 0 24 24" aria-hidden>
        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
        <path
          className="opacity-75"
          fill="currentColor"
          d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"
        />
      </svg>
      {label ? <span className="ml-2 text-sm">{label}</span> : asStatus ? <span className="sr-only">Loading</span> : null}
    </span>
  )
}
