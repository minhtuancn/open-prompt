import ReactMarkdown from 'react-markdown'
import rehypeSanitize from 'rehype-sanitize'

interface Props {
  text: string
}

/** MarkdownRenderer render Markdown an toàn với react-markdown + rehype-sanitize */
export function MarkdownRenderer({ text }: Props) {
  return (
    <div className="markdown-body text-sm text-white/90 leading-relaxed">
      <ReactMarkdown
        rehypePlugins={[rehypeSanitize]}
        components={{
          h1: ({ children }) => (
            <h1 className="text-white font-bold mt-3 mb-1">{children}</h1>
          ),
          h2: ({ children }) => (
            <h2 className="text-white font-semibold mt-3 mb-1">{children}</h2>
          ),
          h3: ({ children }) => (
            <h3 className="text-white font-semibold text-sm mt-3 mb-1">{children}</h3>
          ),
          p: ({ children }) => (
            <p className="mb-2">{children}</p>
          ),
          code: ({ className, children, ...props }) => {
            const isBlock = className?.startsWith('language-')
            if (isBlock) {
              return (
                <pre className="bg-black/30 rounded p-3 my-2 overflow-x-auto">
                  <code className="text-xs font-mono text-indigo-300" {...props}>
                    {children}
                  </code>
                </pre>
              )
            }
            return (
              <code className="bg-white/10 px-1 py-0.5 rounded text-xs font-mono text-indigo-300" {...props}>
                {children}
              </code>
            )
          },
          strong: ({ children }) => (
            <strong className="text-white font-semibold">{children}</strong>
          ),
          em: ({ children }) => (
            <em className="text-white/80 italic">{children}</em>
          ),
          ul: ({ children }) => (
            <ul className="list-disc list-inside mb-2 space-y-1">{children}</ul>
          ),
          ol: ({ children }) => (
            <ol className="list-decimal list-inside mb-2 space-y-1">{children}</ol>
          ),
          li: ({ children }) => (
            <li className="text-white/80">{children}</li>
          ),
          blockquote: ({ children }) => (
            <blockquote className="border-l-2 border-white/20 pl-3 my-2 text-white/60">{children}</blockquote>
          ),
        }}
      >
        {text}
      </ReactMarkdown>
    </div>
  )
}
