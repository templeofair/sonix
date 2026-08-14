import { Link, useSearchParams } from 'react-router-dom'
import { catalog } from '../catalog'

/** Index of mounts + optional state chips from catalog. */
export default function CatalogPage() {
  return (
    <div className="space-y-6 max-w-3xl">
      <div>
        <h1 className="text-xl font-semibold text-gray-900 tracking-tight">Catalog</h1>
        <p className="text-sm text-muted mt-1">
          Synced mounts use real Sonix UI + fixtures. Start mock UX with RESET, then WORK under{' '}
          <code className="text-xs">experiments/</code>.
        </p>
      </div>
      <ul className="space-y-3">
        {catalog.map((entry) => (
          <li key={entry.id}>
            <Link
              to={`/__ui/_kit/${entry.path}`}
              className="block rounded-card border border-border bg-card shadow-card p-4 hover:border-accent/30 hover:shadow-md transition-all focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/35"
            >
              <div className="flex items-baseline justify-between gap-2">
                <span className="font-medium text-gray-900">{entry.title}</span>
                <span className="text-[10px] uppercase tracking-wider text-muted-subtle shrink-0">
                  {entry.status}
                </span>
              </div>
              <p className="text-sm text-muted mt-1">{entry.blurb}</p>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  )
}

/** Renders state switcher links for the current mount. */
export function StateSwitcher({
  states,
  param = 'state',
}: {
  states: string[]
  param?: string
}) {
  const [sp] = useSearchParams()
  const current = sp.get(param) || states[0]
  return (
    <div className="flex flex-wrap gap-2 mb-4" role="group" aria-label="Mock state">
      {states.map((s) => {
        const next = new URLSearchParams(sp)
        next.set(param, s)
        const active = current === s
        return (
          <Link
            key={s}
            to={`?${next.toString()}`}
            className={`control inline-flex px-3 py-1.5 rounded-btn text-xs font-medium border transition-colors focus-visible:ring-2 focus-visible:ring-accent/35 ${
              active
                ? 'border-accent bg-accent/10 text-accent'
                : 'border-border bg-white text-muted hover:bg-surface'
            }`}
          >
            {s}
          </Link>
        )
      })}
    </div>
  )
}
