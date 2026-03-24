import { useState } from 'react'
import { useAuthStore } from '../../store/authStore'

interface PromptInput {
  id?: number
  title: string
  content: string
  category: string
  tags: string
  is_slash: boolean
  slash_name: string
}

interface Props {
  initial?: PromptInput
  onSave: (prompt: PromptInput) => void
  onCancel: () => void
}

export function PromptEditor({ initial, onSave, onCancel }: Props) {
  const token = useAuthStore((s) => s.token)
  const [form, setForm] = useState<PromptInput>(initial ?? {
    title: '',
    content: '',
    category: '',
    tags: '',
    is_slash: false,
    slash_name: '',
  })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    if (!form.title.trim() || !form.content.trim()) {
      setError('Title và content không được để trống')
      return
    }
    if (form.is_slash && !/^[a-z0-9_-]{1,32}$/.test(form.slash_name)) {
      setError('Slash name chỉ được chứa a-z, 0-9, - và _ (tối đa 32 ký tự)')
      return
    }

    if (!token) {
      setError('Chưa đăng nhập')
      return
    }

    setSaving(true)
    try {
      const method = form.id ? 'prompts.update' : 'prompts.create'
      const params = form.id
        ? { token, id: form.id, ...form }
        : { token, ...form }
      await window.__rpc?.call(method, params)
      onSave(form)
    } catch (e) {
      setError(String(e))
    } finally {
      setSaving(false)
    }
  }

  const field = (key: keyof PromptInput) => ({
    value: form[key] as string,
    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      setForm(prev => ({ ...prev, [key]: e.target.value })),
  })

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="text-xs text-white/50 mb-1 block">Title *</label>
        <input
          type="text"
          {...field('title')}
          placeholder="Tên prompt"
          className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/30 outline-none focus:border-indigo-500/50"
        />
      </div>

      <div>
        <label className="text-xs text-white/50 mb-1 block">Content * (dùng {'{{.input}}'} cho nội dung người dùng nhập)</label>
        <textarea
          {...field('content')}
          placeholder="Write an email about {{.input}}..."
          rows={6}
          className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/30 outline-none focus:border-indigo-500/50 resize-none font-mono"
        />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="text-xs text-white/50 mb-1 block">Category</label>
          <input
            type="text"
            {...field('category')}
            placeholder="productivity"
            className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/30 outline-none focus:border-indigo-500/50"
          />
        </div>
        <div>
          <label className="text-xs text-white/50 mb-1 block">Tags (phân cách bằng dấu phẩy)</label>
          <input
            type="text"
            {...field('tags')}
            placeholder="email,writing"
            className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/30 outline-none focus:border-indigo-500/50"
          />
        </div>
      </div>

      <div className="flex items-center gap-3">
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={form.is_slash}
            onChange={e => setForm(prev => ({ ...prev, is_slash: e.target.checked }))}
            className="w-4 h-4 rounded"
          />
          <span className="text-sm text-white/70">Slash command</span>
        </label>
        {form.is_slash && (
          <input
            type="text"
            {...field('slash_name')}
            placeholder="email"
            className="flex-1 bg-white/5 border border-white/10 rounded-lg px-3 py-1.5 text-sm text-white placeholder-white/30 outline-none focus:border-indigo-500/50 font-mono"
          />
        )}
      </div>

      {error && <p className="text-red-400 text-xs">{error}</p>}

      <div className="flex gap-3 justify-end">
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 text-sm text-white/50 hover:text-white transition-colors"
        >
          Huỷ
        </button>
        <button
          type="submit"
          disabled={saving}
          className="px-4 py-2 bg-indigo-500/80 hover:bg-indigo-500 text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
        >
          {saving ? 'Đang lưu...' : form.id ? 'Cập nhật' : 'Tạo mới'}
        </button>
      </div>
    </form>
  )
}
