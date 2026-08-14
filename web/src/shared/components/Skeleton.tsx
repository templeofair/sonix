type Props = {
  className?: string
}

/** Placeholder block for loading states — matches card rhythm without jumping layout. */
export default function Skeleton({ className = 'h-24 w-full' }: Props) {
  return (
    <div
      className={`rounded-card bg-border/60 animate-pulse ${className}`}
      aria-hidden
    />
  )
}
