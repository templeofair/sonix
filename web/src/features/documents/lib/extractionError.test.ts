import { describe, it, expect } from 'vitest'
import { summarizeExtractionError } from './extractionError'

describe('summarizeExtractionError', () => {
  it('maps context size errors to a short summary with detail', () => {
    const raw =
      'LLM vision: ollama 400 Bad Request: {"error":{"message":"request (4452 tokens) exceeds the available context size (4096 tokens)","type":"exceed_context_size_error"}}'
    const { summary, detail } = summarizeExtractionError(raw)
    expect(summary).toMatch(/too large for the AI model/i)
    expect(summary).not.toMatch(/4452/)
    expect(detail).toBeNull()
  })

  it('maps connection refused', () => {
    const { summary, detail } = summarizeExtractionError('connection refused')
    expect(summary).toMatch(/Could not reach Ollama/i)
    expect(detail).toBeNull()
  })

  it('does not echo paths or URLs', () => {
    const { summary, detail } = summarizeExtractionError(
      'save original text: open /app/data/uploads/12/page_0.png: no such file',
    )
    expect(summary).toMatch(/Check Ollama/i)
    expect(summary).not.toMatch(/uploads/)
    expect(detail).toBeNull()
  })

  it('handles empty', () => {
    const { summary, detail } = summarizeExtractionError('')
    expect(summary).toMatch(/Check Ollama/i)
    expect(detail).toBeNull()
  })
})
