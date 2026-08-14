/** Pill styles for document `status` in list UIs (inbox / year / search / pending). */
export function documentStatusPillClass(status: string): string {
  switch (status) {
    case 'ready':
      return 'bg-success-soft text-success border-success-border/80'
    case 'processing':
      return 'bg-warning-soft text-warning border-warning-border/80'
    case 'failed':
      return 'bg-danger-soft text-danger border-danger-border/80'
    case 'partial':
      return 'bg-orange-50 text-orange-900 border-orange-200/80'
    case 'pending':
      return 'bg-surface text-muted border-border'
    default:
      return 'bg-surface text-muted border-border'
  }
}
