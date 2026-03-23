import { useMemo } from 'react'

interface Props {
  text: string
}

/** MarkdownRenderer render Markdown cơ bản (code blocks, bold, italic, inline code) */
export function MarkdownRenderer({ text }: Props) {
  const rendered = useMemo(() => renderMarkdown(text), [text])
  return <div className="text-white/90 text-sm leading-relaxed" dangerouslySetInnerHTML={{ __html: rendered }} />
}

function renderMarkdown(text: string): string {
  const lines = text.split('\n')
  const result: string[] = []
  let inCodeBlock = false
  let codeLines: string[] = []
  for (const line of lines) {
    if (line.startsWith('```')) {
      if (inCodeBlock) {
        const code = escapeHtml(codeLines.join('\n'))
        result.push(`<pre class="bg-black/30 border border-white/10 rounded-lg p-3 my-2 overflow-x-auto"><code class="text-xs font-mono text-green-300">${code}</code></pre>`)
        codeLines = []
        inCodeBlock = false
      } else {
        inCodeBlock = true
      }
      continue
    }

    if (inCodeBlock) {
      codeLines.push(line)
      continue
    }

    // Headings
    let processed = line
    if (processed.startsWith('### ')) {
      processed = `<h3 class="text-white font-semibold text-sm mt-3 mb-1">${processed.slice(4)}</h3>`
    } else if (processed.startsWith('## ')) {
      processed = `<h2 class="text-white font-semibold mt-3 mb-1">${processed.slice(3)}</h2>`
    } else if (processed.startsWith('# ')) {
      processed = `<h1 class="text-white font-bold mt-3 mb-1">${processed.slice(2)}</h1>`
    } else if (processed.startsWith('- ') || processed.startsWith('* ')) {
      processed = `<li class="ml-4 list-disc">${inlineFormat(processed.slice(2))}</li>`
    } else if (/^\d+\.\s/.test(processed)) {
      const content = processed.replace(/^\d+\.\s/, '')
      processed = `<li class="ml-4 list-decimal">${inlineFormat(content)}</li>`
    } else {
      processed = inlineFormat(processed)
      if (processed.trim()) {
        processed = `<p class="my-0.5">${processed}</p>`
      } else {
        processed = '<br/>'
      }
    }
    result.push(processed)
  }

  // Code block không đóng
  if (inCodeBlock && codeLines.length > 0) {
    const code = escapeHtml(codeLines.join('\n'))
    result.push(`<pre class="bg-black/30 border border-white/10 rounded-lg p-3 my-2 overflow-x-auto"><code class="text-xs font-mono text-green-300">${code}</code></pre>`)
  }

  return result.join('\n')
}

function inlineFormat(text: string): string {
  let result = escapeHtml(text)
  // Inline code
  result = result.replace(/`([^`]+)`/g, '<code class="bg-white/10 px-1 py-0.5 rounded text-xs font-mono text-indigo-300">$1</code>')
  // Bold
  result = result.replace(/\*\*([^*]+)\*\*/g, '<strong class="text-white font-semibold">$1</strong>')
  // Italic
  result = result.replace(/\*([^*]+)\*/g, '<em class="text-white/80 italic">$1</em>')
  return result
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}
