import type { ReactNode } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { normalizeMarkdownTables } from '../lib/markdownNormalize'

type Props = {
  text: string
  className?: string
}

type MdProps = { children?: ReactNode; href?: string }

/** Strip a single outer ``` / ```markdown fence if the whole body is wrapped. */
function stripOuterFence(md: string): string {
  const trimmed = md.trim()
  const m = trimmed.match(/^```(?:markdown|md|gfm)?\s*\n([\s\S]*?)\n```\s*$/i)
  return m ? m[1] : md
}

/** Prepare model Markdown for display: unwrap fences, fix orphan tables, surface [unleserlich]. */
export function prepareExtractionMarkdown(text: string): string {
  let t = stripOuterFence(text ?? '')
  t = normalizeMarkdownTables(t)
  // Vision models (turbo) mark illegible spots; render as inline code so they
  // stand out without looking like broken brackets.
  t = t.replace(/\[unleserlich\]/gi, '`[unleserlich]`')
  return t
}

// MarkdownText renders extracted document text as Markdown.
//
// - GFM on; rehype-raw off (untrusted OCR/LLM text must not run HTML).
// - Explicit element styles (no @tailwindcss/typography).
// - break-words on the wrapper and table cells so long barcode/ID lines
//   from German letters do not overflow the modal horizontally.
export default function MarkdownText({ text, className }: Props) {
  const normalized = prepareExtractionMarkdown(text)
  return (
    <div
      className={`text-gray-800 text-sm leading-relaxed space-y-3 break-words [overflow-wrap:anywhere] ${className ?? ''}`}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          h1: ({ children }: MdProps) => <h1 className="text-lg font-semibold text-gray-900 mt-2">{children}</h1>,
          h2: ({ children }: MdProps) => <h2 className="text-base font-semibold text-gray-900 mt-2">{children}</h2>,
          h3: ({ children }: MdProps) => <h3 className="text-sm font-semibold text-gray-900 mt-2">{children}</h3>,
          p: ({ children }: MdProps) => <p className="whitespace-pre-wrap break-words">{children}</p>,
          ul: ({ children }: MdProps) => <ul className="list-disc list-inside space-y-1 pl-2">{children}</ul>,
          ol: ({ children }: MdProps) => <ol className="list-decimal list-inside space-y-1 pl-2">{children}</ol>,
          li: ({ children }: MdProps) => <li className="leading-relaxed break-words">{children}</li>,
          strong: ({ children }: MdProps) => <strong className="font-semibold text-gray-900">{children}</strong>,
          em: ({ children }: MdProps) => <em className="italic">{children}</em>,
          a: ({ children, href }: MdProps) => (
            <a
              href={href}
              target="_blank"
              rel="noopener noreferrer"
              className="text-accent underline hover:text-accent-light break-all"
            >
              {children}
            </a>
          ),
          code: ({ children }: MdProps) => (
            <code className="px-1 py-0.5 rounded bg-gray-100 text-gray-800 font-mono text-xs break-all">
              {children}
            </code>
          ),
          pre: ({ children }: MdProps) => (
            <pre className="p-3 rounded-lg bg-gray-50 border border-gray-200 overflow-x-auto text-xs font-mono">{children}</pre>
          ),
          blockquote: ({ children }: MdProps) => (
            <blockquote className="border-l-4 border-gray-200 pl-3 text-gray-600 italic">{children}</blockquote>
          ),
          hr: () => <hr className="border-gray-200" />,
          table: ({ children }: MdProps) => (
            <div className="overflow-x-auto -mx-1">
              <table className="min-w-full border-collapse text-xs">{children}</table>
            </div>
          ),
          thead: ({ children }: MdProps) => <thead className="bg-gray-50">{children}</thead>,
          tbody: ({ children }: MdProps) => <tbody>{children}</tbody>,
          tr: ({ children }: MdProps) => <tr className="border-b border-gray-200 last:border-b-0">{children}</tr>,
          th: ({ children }: MdProps) => (
            <th className="px-2 py-1.5 text-left font-semibold text-gray-700 border border-gray-200 break-words">
              {children}
            </th>
          ),
          td: ({ children }: MdProps) => (
            <td className="px-2 py-1.5 align-top border border-gray-200 break-words">{children}</td>
          ),
        }}
      >
        {normalized}
      </ReactMarkdown>
    </div>
  )
}
