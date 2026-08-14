import { COLOUR_MODE_LABELS, COLOUR_MODES, type ColourMode } from '../lib/colourMode'

type Props = {
  mode: ColourMode
  onModeChange: (mode: ColourMode) => void
  /** Dark camera chrome vs light review surface. */
  variant?: 'dark' | 'light'
  className?: string
}

export default function ColourModeControls({
  mode,
  onModeChange,
  variant = 'dark',
  className = '',
}: Props) {
  const dark = variant === 'dark'
  const segmentIdle = dark
    ? 'text-white/70 hover:bg-white/10'
    : 'text-gray-700 hover:bg-surface'
  const segmentActive = dark ? 'bg-white text-gray-900 shadow-sm' : 'bg-accent text-white shadow-sm'

  return (
    <div
      role="group"
      aria-label="Colour mode"
      className={`flex rounded-btn p-0.5 gap-0.5 ${dark ? 'bg-white/10' : 'bg-surface border border-border'} ${className}`}
    >
      {COLOUR_MODES.map((m) => (
        <button
          key={m}
          type="button"
          aria-pressed={mode === m}
          onClick={() => onModeChange(m)}
          className={`flex-1 min-h-[44px] px-3 rounded-[calc(var(--sonix-radius-btn)-2px)] text-sm font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/50 ${
            mode === m ? segmentActive : segmentIdle
          }`}
        >
          {COLOUR_MODE_LABELS[m]}
        </button>
      ))}
    </div>
  )
}
