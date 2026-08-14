export type CatalogStatus = 'synced' | 'experimenting' | 'deferred'

export type CatalogEntry = {
  id: string
  title: string
  path: string
  status: CatalogStatus
  blurb: string
  /** Query ?state= values when supported */
  states?: string[]
}

export const catalog: CatalogEntry[] = [
  {
    id: 'primitives',
    title: 'Primitives',
    path: 'primitives',
    status: 'synced',
    blurb: 'Button, Banner, Field, Card, EmptyState, Skeleton, Spinner',
  },
  {
    id: 'cards',
    title: 'Document cards',
    path: 'cards',
    status: 'synced',
    blurb: 'Real DocumentCard · grid/list via ?layout=',
    states: ['grid', 'list'],
  },
  {
    id: 'login',
    title: 'Login',
    path: 'login',
    status: 'synced',
    blurb: 'Real LoginForm + MockAuthProvider',
  },
  {
    id: 'library',
    title: 'Library',
    path: 'library',
    status: 'synced',
    blurb: 'Composed leaves · ?state=populated|empty|loading',
    states: ['populated', 'empty', 'loading'],
  },
  {
    id: 'detail',
    title: 'Detail / AI states',
    path: 'detail',
    status: 'synced',
    blurb: 'Status stand-in · ?state=pending|processing|failed|partial|ready',
    states: ['pending', 'processing', 'failed', 'partial', 'ready'],
  },
  {
    id: 'settings',
    title: 'Settings',
    path: 'settings',
    status: 'synced',
    blurb: 'Fixture settings form (real Field/Card)',
  },
  {
    id: 'scan',
    title: 'Scan hub',
    path: 'scan',
    status: 'synced',
    blurb: 'Camera / upload chooser shape',
  },
  {
    id: 'camera',
    title: 'Camera',
    path: 'camera',
    status: 'deferred',
    blurb: 'Device-only placeholder',
  },
]
