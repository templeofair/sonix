import { strict as assert } from 'node:assert'
import { normalizeMarkdownTables } from './markdownNormalize'

// Lightweight test file. We don't have a frontend test runner wired up,
// but this is pure TypeScript with no React deps so it can be executed
// via `npx tsx web/src/lib/markdownNormalize.test.ts` (or equivalent)
// when one is added. Every test is a single-function assert so the
// failure mode is obvious.

function test(name: string, fn: () => void) {
  try {
    fn()
    console.log(`ok  ${name}`)
  } catch (e) {
    console.error(`FAIL ${name}`)
    throw e
  }
}

test('empty input returns empty', () => {
  assert.equal(normalizeMarkdownTables(''), '')
})

test('plain prose is unchanged', () => {
  const s = 'Hello world.\nSecond line.\n'
  assert.equal(normalizeMarkdownTables(s), s)
})

test('valid GFM table is unchanged', () => {
  const s = ['| A | B |', '| --- | --- |', '| 1 | 2 |'].join('\n')
  assert.equal(normalizeMarkdownTables(s), s)
})

test('orphan two-row block gets a separator after the first row', () => {
  const input = ['| Total | 4.11 € |', '| VAT | 0.29 € |'].join('\n')
  const expected = ['| Total | 4.11 € |', '| --- | --- |', '| VAT | 0.29 € |'].join('\n')
  assert.equal(normalizeMarkdownTables(input), expected)
})

test('orphan single row becomes a header-only table', () => {
  const input = '| Total | 4.40 € |'
  const expected = ['| Total | 4.40 € |', '| --- | --- |'].join('\n')
  assert.equal(normalizeMarkdownTables(input), expected)
})

test('inline pipe in prose is NOT promoted to a table', () => {
  // Line does not start with `|`, so it's not a pipe-row.
  const s = 'Total (gross) | 4.40 € |'
  assert.equal(normalizeMarkdownTables(s), s)
})

test('function is idempotent', () => {
  const input = ['| A | B |', '| C | D |', '', '| X | Y |'].join('\n')
  const once = normalizeMarkdownTables(input)
  const twice = normalizeMarkdownTables(once)
  assert.equal(twice, once)
})

test('mixed valid table + orphan block in same document', () => {
  const input = [
    '| H1 | H2 |',
    '| --- | --- |',
    '| d1 | d2 |',
    '',
    '| Total | 4.40 |',
    '| VAT | 0.29 |',
  ].join('\n')
  const expected = [
    '| H1 | H2 |',
    '| --- | --- |',
    '| d1 | d2 |',
    '',
    '| Total | 4.40 |',
    '| --- | --- |',
    '| VAT | 0.29 |',
  ].join('\n')
  assert.equal(normalizeMarkdownTables(input), expected)
})

test('separator width matches header cell count', () => {
  const input = '| A | B | C | D |'
  const expected = ['| A | B | C | D |', '| --- | --- | --- | --- |'].join('\n')
  assert.equal(normalizeMarkdownTables(input), expected)
})

test('lone separator without a header is left alone', () => {
  const s = '| --- | --- |'
  // We don't start a block on a lone separator; it passes through.
  assert.equal(normalizeMarkdownTables(s), s)
})

console.log('\nAll markdownNormalize tests passed.')
