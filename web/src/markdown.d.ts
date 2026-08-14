// Fallback declarations for react-markdown and remark-gfm. The real
// packages are installed inside the Docker build (see web/package.json);
// this lets the IDE's local TypeScript resolve the imports without
// node_modules being populated on the host. Only the surface we use
// today is declared — extend if you start using more of the API.
declare module 'react-markdown' {
  import type { ComponentType, ReactNode } from 'react'

  type MarkdownComponentProps = {
    children?: ReactNode
    href?: string
    [key: string]: unknown
  }

  export type Components = {
    [tag: string]: ComponentType<MarkdownComponentProps>
  }

  export interface ReactMarkdownProps {
    children?: string
    remarkPlugins?: unknown[]
    rehypePlugins?: unknown[]
    components?: Components
  }

  const ReactMarkdown: ComponentType<ReactMarkdownProps>
  export default ReactMarkdown
}

declare module 'remark-gfm' {
  const remarkGfm: unknown
  export default remarkGfm
}
