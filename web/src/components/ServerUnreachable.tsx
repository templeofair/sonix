type Props = {
  onRetry: () => void
  busy?: boolean
}

/** Full-page state when the API host cannot be reached (LAN drop, server down). */
export default function ServerUnreachable({ onRetry, busy }: Props) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-surface px-4 py-10">
      <div className="w-full max-w-sm rounded-card border border-border bg-card shadow-card p-6 sm:p-8 text-center space-y-4">
        <div
          className="w-12 h-12 mx-auto rounded-full bg-amber-100 flex items-center justify-center"
          aria-hidden
        >
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#b45309" strokeWidth="2" strokeLinecap="round">
            <path d="M12 9v4M12 17h.01" />
            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
          </svg>
        </div>
        <h1 className="text-lg font-semibold text-gray-900 tracking-tight">Cannot reach Sonix</h1>
        <p className="text-sm text-muted leading-relaxed">
          The app could not connect to the server. Check that you are on the same network and that Sonix is running, then try again.
        </p>
        <button
          type="button"
          onClick={onRetry}
          disabled={busy}
          className="control w-full px-4 py-2.5 bg-accent text-white rounded-btn font-medium text-sm shadow-sm hover:opacity-95 disabled:opacity-50 transition-opacity focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/40 focus-visible:ring-offset-2"
        >
          {busy ? 'Trying…' : 'Try again'}
        </button>
      </div>
    </div>
  )
}
