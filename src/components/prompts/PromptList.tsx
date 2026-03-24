import { useEffect, useState } from 'react'
import { useAuthStore } from '../../store/authStore'

interface Prompt {
  id: number
  title: string
  content: string
  category: string
  tags: string
  is_slash: boolean
  slash_name: string
}

interface Props {
  onEdit: (prompt: Prompt) => void
}

export function PromptList({ onEdit }: Props) {
  const token = useAuthStore((s) => s.token)
  const [prompts, setPrompts] = useState<Prompt[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const loadPrompts = async () => {
    if (!token) return
    try {
      const res = await window.__rpc?.call('prompts.list', { token })
      const data = res as { prompts: Prompt[] }
      setPrompts(data?.prompts || [])
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { loadPrompts() }, [])

  const handleDelete = async (id: number) => {
    if (!confirm('Xoá prompt này?')) return
    if (!token) return
    try {
      await window.__rpc?.call('prompts.delete', { token, id })
      setPrompts(prev => prev.filter(p => p.id !== id))
    } catch (e) {
      alert(`Lỗi xoá: ${e}`)
    }
  }

  if (loading) return <div className="text-white/50 text-sm p-4">Đang tải...</div>
  if (error) return <div className="text-red-400 text-sm p-4">{error}</div>

  return (
    <div className="space-y-2">
      {prompts.length === 0 && (
        <div className="text-white/40 text-sm text-center py-8">
          Chưa có prompt nào. Tạo mới để bắt đầu.
        </div>
      )}
      {prompts.map(prompt => (
        <div
          key={prompt.id}
          className="flex items-start gap-3 p-3 bg-white/5 rounded-lg border border-white/10 hover:border-white/20 transition-colors"
        >
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-white truncate">{prompt.title}</span>
              {prompt.is_slash && (
                <span className="font-mono text-xs text-indigo-400 bg-indigo-500/20 px-1.5 py-0.5 rounded">
                  /{prompt.slash_name}
                </span>
              )}
            </div>
            <p className="text-xs text-white/40 mt-0.5 line-clamp-2">{prompt.content}</p>
            {prompt.tags && (
              <div className="flex gap-1 mt-1">
                {prompt.tags.split(',').map(tag => (
                  <span key={tag} className="text-xs text-white/30 bg-white/5 px-1.5 py-0.5 rounded">
                    {tag.trim()}
                  </span>
                ))}
              </div>
            )}
          </div>
          <div className="flex gap-1 flex-shrink-0">
            <button
              onClick={() => onEdit(prompt)}
              className="text-xs text-white/40 hover:text-white px-2 py-1 rounded transition-colors"
            >
              Sửa
            </button>
            <button
              onClick={() => handleDelete(prompt.id)}
              className="text-xs text-red-400/60 hover:text-red-400 px-2 py-1 rounded transition-colors"
            >
              Xoá
            </button>
          </div>
        </div>
      ))}
    </div>
  )
}
