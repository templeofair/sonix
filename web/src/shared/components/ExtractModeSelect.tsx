import type { ChangeEvent } from 'react'
import { fieldInputClass } from './Field'

export type ExtractMode = 'ocr' | 'llm'

/** Map persisted extractions.engine_id to a mode, or null if unknown / missing. */
export function extractModeFromEngineId(engineId?: string | null): ExtractMode | null {
  const id = (engineId ?? '').trim().toLowerCase()
  if (!id) return null
  if (id.startsWith('tesseract')) return 'ocr'
  if (id.startsWith('vision')) return 'llm'
  return null
}

type Props = {
  id: string
  value: ExtractMode
  onChange: (next: ExtractMode) => void
  /** Amber styling for failed / partial panels. */
  tone?: 'neutral' | 'amber'
  disabled?: boolean
  className?: string
}

/** Combobox for OCR (Tesseract) vs LLM vision — Settings and document extract/re-process. */
export default function ExtractModeSelect({
  id,
  value,
  onChange,
  tone = 'neutral',
  disabled = false,
  className = '',
}: Props) {
  const labelClass = tone === 'amber' ? 'text-amber-900' : 'text-muted'
  return (
    <div className={className}>
      <label htmlFor={id} className={`block text-xs font-semibold uppercase tracking-wider mb-2 ${labelClass}`}>
        Extraction mode
      </label>
      <select
        id={id}
        value={value}
        disabled={disabled}
        onChange={(e: ChangeEvent<HTMLSelectElement>) =>
          onChange(e.target.value === 'ocr' ? 'ocr' : 'llm')
        }
        className={`control min-h-[44px] ${fieldInputClass}${
          tone === 'amber' ? ' border-amber-300 bg-amber-50/50' : ''
        }`}
      >
        <option value="ocr">OCR (Tesseract)</option>
        <option value="llm">LLM vision</option>
      </select>
    </div>
  )
}
