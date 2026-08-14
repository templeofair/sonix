/** Map extraction_error strings to a short user-facing summary. Never echo raw Go/Ollama dumps. */
export function summarizeExtractionError(raw: string | undefined | null): {
  summary: string
  detail: string | null
} {
  const text = raw?.trim() || ''
  if (!text) {
    return {
      summary: 'Extraction failed. Check Ollama in Settings, then retry.',
      detail: null,
    }
  }

  const lower = text.toLowerCase()

  if (
    lower.includes('exceed_context_size') ||
    lower.includes('exceeds the available context') ||
    (lower.includes('n_prompt_tokens') && lower.includes('n_ctx')) ||
    lower.includes('too large for the ai model')
  ) {
    return {
      summary: 'This page is too large for the AI model’s context. Try OCR, or retry extraction.',
      detail: null,
    }
  }

  if (
    lower.includes('connection refused') ||
    lower.includes('connect:') ||
    lower.includes('no such host') ||
    lower.includes('could not reach ollama')
  ) {
    return {
      summary: 'Could not reach Ollama. Check the URL in Settings.',
      detail: null,
    }
  }

  if (lower.includes('timeout') || lower.includes('deadline exceeded') || lower.includes('timed out')) {
    return {
      summary: 'Ollama timed out. Try again, or use a smaller model.',
      detail: null,
    }
  }

  if (lower.includes('404') && lower.includes('model')) {
    return {
      summary: 'Ollama model not found. Check the model name in Settings.',
      detail: null,
    }
  }

  let brief = text
  brief = brief.replace(/^LLM vision:\s*/i, '')
  brief = brief.replace(/^ollama\s+\d+\s+[^:]+:\s*/i, '')
  if (brief.length > 160) {
    brief = brief.slice(0, 157).trimEnd() + '…'
  }
  const looksInternal =
    brief.includes('/') ||
    brief.includes('\\') ||
    brief.includes('http://') ||
    brief.includes('https://') ||
    brief.includes('{')
  if (looksInternal) {
    return {
      summary: 'Extraction failed. Check Ollama in Settings, then retry.',
      detail: null,
    }
  }
  return { summary: brief || 'Extraction failed. Retry or check Settings.', detail: null }
}
