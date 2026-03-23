import { useState } from 'react'
import { callEngine } from '../../hooks/useEngine'

interface Skill {
  id?: number
  name: string
  prompt_text: string
  model: string
  provider: string
  tags: string
}

interface Props {
  skill?: Skill
  onSave: () => void
  onCancel: () => void
}

const PROVIDERS = ['anthropic', 'openai', 'ollama']
const MODELS: Record<string, string[]> = {
  anthropic: ['claude-3-5-sonnet-20241022', 'claude-3-haiku-20240307'],
  openai: ['gpt-4o', 'gpt-4o-mini'],
  ollama: ['llama3.2', 'mistral'],
}

export function SkillEditor({ skill, onSave, onCancel }: Props) {
  const [name, setName] = useState(skill?.name ?? '')
  const [promptText, setPromptText] = useState(skill?.prompt_text ?? '')
  const [provider, setProvider] = useState(skill?.provider ?? 'anthropic')
  const [model, setModel] = useState(skill?.model ?? '')
  const [tags, setTags] = useState(skill?.tags ?? '')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const handleSave = async () => {
    if (!name.trim()) { setError('Tên skill không được rỗng'); return }
    const token = localStorage.getItem('auth_token')
    if (!token) return
    setSaving(true)
    setError('')
    try {
      const payload = { token, name: name.trim(), prompt_text: promptText, provider, model, tags }
      if (skill?.id) {
        await callEngine('skills.update', { ...payload, id: skill.id })
      } else {
        await callEngine('skills.create', payload)
      }
      onSave()
    } catch (e) {
      setError(String(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <h3 className="text-sm font-semibold text-white">{skill?.id ? 'Sửa skill' : 'Tạo skill mới'}</h3>

      <div>
        <label className="text-xs text-white/50 mb-1 block">Tên skill *</label>
        <input value={name} onChange={(e) => setName(e.target.value)} placeholder="VD: Dịch thuật..." className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50" />
      </div>

      <div>
        <label className="text-xs text-white/50 mb-1 block">Nội dung prompt</label>
        <textarea value={promptText} onChange={(e) => setPromptText(e.target.value)} placeholder="Bạn là trợ lý {{role}}..." rows={4} className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50 resize-none font-mono" />
        <p className="text-xs text-white/30 mt-1">Dùng {`{{variable}}`} để tạo biến động</p>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div>
          <label className="text-xs text-white/50 mb-1 block">Provider</label>
          <select value={provider} onChange={(e) => { setProvider(e.target.value); setModel('') }} className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white outline-none focus:border-indigo-500/50">
            {PROVIDERS.map((p) => <option key={p} value={p} className="bg-gray-900">{p}</option>)}
          </select>
        </div>
        <div>
          <label className="text-xs text-white/50 mb-1 block">Model</label>
          <select value={model} onChange={(e) => setModel(e.target.value)} className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white outline-none focus:border-indigo-500/50">
            <option value="" className="bg-gray-900">Mặc định</option>
            {(MODELS[provider] ?? []).map((m) => <option key={m} value={m} className="bg-gray-900">{m}</option>)}
          </select>
        </div>
      </div>

      <div>
        <label className="text-xs text-white/50 mb-1 block">Tags (phân cách bằng dấu phẩy)</label>
        <input value={tags} onChange={(e) => setTags(e.target.value)} placeholder="dịch thuật, code, viết lách" className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50" />
      </div>

      {error && <p className="text-xs text-red-400">{error}</p>}

      <div className="flex gap-2 justify-end">
        <button onClick={onCancel} className="text-sm px-4 py-2 text-white/50 hover:text-white transition-colors">Hủy</button>
        <button onClick={handleSave} disabled={saving} className="text-sm px-4 py-2 bg-indigo-500/80 hover:bg-indigo-500 text-white rounded-lg transition-colors disabled:opacity-50">
          {saving ? 'Đang lưu...' : 'Lưu'}
        </button>
      </div>
    </div>
  )
}
