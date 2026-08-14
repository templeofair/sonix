import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import MarkdownText, { prepareExtractionMarkdown } from './MarkdownText'

describe('prepareExtractionMarkdown', () => {
  it('inserts a GFM separator for orphan pipe rows', () => {
    const out = prepareExtractionMarkdown('| Total | 4.11 € |\n| VAT | 0.29 € |')
    expect(out).toContain('| --- | --- |')
  })

  it('marks [unleserlich] as inline code for visibility', () => {
    expect(prepareExtractionMarkdown('Name: [unleserlich]')).toBe('Name: `[unleserlich]`')
  })

  it('unwraps a whole-document markdown fence', () => {
    const fenced = '```markdown\n# Title\n\nBody\n```'
    expect(prepareExtractionMarkdown(fenced)).toBe('# Title\n\nBody')
  })
})

describe('MarkdownText', () => {
  it('renders bold Markdown', () => {
    render(<MarkdownText text="Hello **world**" />)
    expect(screen.getByText('world').tagName).toBe('STRONG')
  })

  it('renders a GFM table from orphan pipe rows', () => {
    const { container } = render(
      <MarkdownText text={'| Total | 4.11 € |\n| VAT | 0.29 € |'} />
    )
    expect(container.querySelector('table')).not.toBeNull()
    expect(screen.getByText('Total')).toBeTruthy()
    expect(screen.getByText('4.11 €')).toBeTruthy()
  })

  it('surfaces unleserlich markers', () => {
    render(<MarkdownText text="Zeile mit [unleserlich] Stelle" />)
    const mark = screen.getByText('[unleserlich]')
    expect(mark.tagName).toBe('CODE')
  })

  it('keeps long reference numbers in the document flow', () => {
    const long = '14 3038 6551 78 1000 1724 DV07.26 1.10 Deutsche Post'
    const { container } = render(<MarkdownText text={long} />)
    expect(container.textContent).toContain(long)
    expect(container.firstElementChild?.className).toMatch(/break-words/)
  })

  it('does not execute raw HTML from the model', () => {
    render(<MarkdownText text={'<script>window.__xss=1</script>safe'} />)
    expect(screen.getByText(/safe/)).toBeTruthy()
    expect(document.querySelector('script')).toBeNull()
  })
})
