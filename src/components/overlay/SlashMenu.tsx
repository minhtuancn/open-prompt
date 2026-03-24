import { useEffect, useMemo, useRef, useState } from 'react'
import Fuse from 'fuse.js'

export interface SlashCommand {
  id: number
  slash_name: string
  title: string
  content: string
  category: string
  tags: string
  required_vars: string[]
}

interface Props {
  commands: SlashCommand[]
  query: string
  onSelect: (cmd: SlashCommand) => void
  onClose: () => void
  visible: boolean
}

const fuseOptions = {
  keys: ['slash_name', 'title', 'tags'],
  threshold: 0.4,
  includeScore: true,
}

export function SlashMenu({ commands, query, onSelect, onClose, visible }: Props) {
  const [activeIndex, setActiveIndex] = useState(0)
  const listRef = useRef<HTMLDivElement>(null)

  const fuse = useMemo(() => new Fuse(commands, fuseOptions), [commands])
  const filtered = useMemo(
    () => query ? fuse.search(query).map(r => r.item) : commands,
    [fuse, query, commands],
  )

  useEffect(() => {
    setActiveIndex(0)
  }, [query])

  useEffect(() => {
    if (!visible) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setActiveIndex(i => Math.min(i + 1, filtered.length - 1))
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setActiveIndex(i => Math.max(i - 1, 0))
      } else if (e.key === 'Enter') {
        e.preventDefault()
        if (filtered[activeIndex]) {
          onSelect(filtered[activeIndex])
        }
      } else if (e.key === 'Escape') {
        onClose()
      }
    }
    window.addEventListener('keydown', handleKeyDown, true)
    return () => window.removeEventListener('keydown', handleKeyDown, true)
  }, [visible, filtered, activeIndex, onSelect, onClose])

  useEffect(() => {
    const container = listRef.current
    if (!container) return
    const activeEl = container.querySelector(`[data-index="${activeIndex}"]`) as HTMLElement
    activeEl?.scrollIntoView({ block: 'nearest' })
  }, [activeIndex])

  if (!visible || filtered.length === 0) return null

  return (
    <div
      className="absolute bottom-full left-0 right-0 mb-1 bg-black/80 backdrop-blur-md border border-white/10 rounded-xl overflow-hidden shadow-2xl z-50 max-h-64 overflow-y-auto"
      ref={listRef}
      role="listbox"
      aria-label="Danh sách lệnh"
    >
      {filtered.map((cmd, index) => (
        <button
          key={cmd.id}
          data-index={index}
          role="option"
          className={`w-full text-left px-4 py-2.5 flex items-center gap-3 transition-colors ${
            index === activeIndex
              ? 'bg-indigo-500/30 text-white'
              : 'text-white/70 hover:bg-white/5'
          }`}
          onMouseEnter={() => setActiveIndex(index)}
          onClick={() => onSelect(cmd)}
        >
          <span className="flex-shrink-0 font-mono text-sm text-indigo-400 bg-indigo-500/20 px-2 py-0.5 rounded-md">
            /{cmd.slash_name}
          </span>
          <span className="flex-1 min-w-0">
            <span className="block text-sm font-medium text-white truncate">{cmd.title}</span>
            {cmd.category && (
              <span className="text-xs text-white/40">{cmd.category}</span>
            )}
          </span>
          {cmd.required_vars.length > 0 && (
            <span className="flex-shrink-0 text-xs text-white/40">
              {cmd.required_vars.map(v => `{${v}}`).join(' ')}
            </span>
          )}
        </button>
      ))}
    </div>
  )
}
