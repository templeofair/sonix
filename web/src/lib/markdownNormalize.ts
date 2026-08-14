// normalizeMarkdownTables rewrites orphan pipe-row blocks into valid GFM
// tables so react-markdown + remark-gfm can render them.
//
// Why this is needed: the metadata/vision LLM is instructed to preserve
// Markdown structure from the source document verbatim. Real-world
// documents (invoices, letters) often contain pipe-delimited summary
// rows that are visually tabular but are not valid GFM, for example:
//
//     | Total (net) 7 % (D) | 4.11 €  |
//     | Including 7 % VAT   | 0.29 €  |
//
// GFM requires a separator line (`| --- | --- |`) between the header
// row and data rows. Without it, remark-gfm renders the whole block as
// plain text with literal pipes. This function detects those orphan
// blocks and inserts a matching separator so the renderer treats them
// as tables. Already-valid tables are left untouched, and non-tabular
// text that merely contains a pipe (e.g. `Total (gross) | 4.40 €`) is
// passed through because it does not match the strict `^\|...\|$`
// pipe-row shape.
//
// Design notes:
// - Pure, no side effects. Safe to call repeatedly on the same input.
// - Idempotent: once separators are injected the output is already a
//   valid table on the next pass.
// - Leading/trailing blank lines are preserved so the surrounding
//   markdown structure (paragraphs, spacing) is unchanged.

// PIPE_ROW: a line that both starts and ends with `|`. This
// intentionally excludes inline-pipe prose like "A | B" or
// "Total (gross) | 4.40 €" which only incidentally contains a pipe.
const PIPE_ROW = /^\s*\|.*\|\s*$/

// SEPARATOR_ROW: a pipe-row whose cells contain only dashes and
// optional alignment colons. Matches GFM shapes like `| --- | --- |`,
// `|:---|---:|:---:|`, or tolerant `| - | - |`.
const SEPARATOR_ROW = /^\s*\|?\s*:?-+:?\s*(?:\|\s*:?-+:?\s*)+\|?\s*$/

function countCells(pipeRow: string): number {
  let s = pipeRow.trim()
  if (s.startsWith('|')) s = s.slice(1)
  if (s.endsWith('|')) s = s.slice(0, -1)
  return s.split('|').length
}

function makeSeparator(cellCount: number): string {
  const parts: string[] = []
  for (let i = 0; i < cellCount; i++) parts.push('---')
  return '| ' + parts.join(' | ') + ' |'
}

export function normalizeMarkdownTables(md: string): string {
  if (!md) return md
  const lines = md.split('\n')
  const out: string[] = []
  let i = 0
  while (i < lines.length) {
    const line = lines[i]
    // A lone separator line (no header above it) is meaningless on its
    // own; don't use it to start a block. Pass through verbatim.
    if (!PIPE_ROW.test(line) || SEPARATOR_ROW.test(line)) {
      out.push(line)
      i++
      continue
    }
    // Gather the contiguous run of pipe-rows / separators that starts
    // here. A blank or non-pipe line terminates the run.
    let end = i
    while (
      end + 1 < lines.length &&
      (PIPE_ROW.test(lines[end + 1]) || SEPARATOR_ROW.test(lines[end + 1]))
    ) {
      end++
    }
    const block = lines.slice(i, end + 1)
    if (block.length >= 2 && SEPARATOR_ROW.test(block[1])) {
      // Valid table (header + separator + 0+ data rows). Keep as-is so
      // any alignment colons the author wrote are preserved.
      out.push(...block)
    } else {
      // Orphan. Treat block[0] as the header, inject a matching-width
      // separator, and keep the remaining rows as data. For a single
      // orphan row this renders as a header-only table, which is still
      // dramatically better than raw `| ... |` text.
      out.push(block[0])
      out.push(makeSeparator(countCells(block[0])))
      if (block.length > 1) out.push(...block.slice(1))
    }
    i = end + 1
  }
  return out.join('\n')
}
