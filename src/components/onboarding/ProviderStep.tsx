import { useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

interface Props {
  onNext: () => void
  onSkip: () => void
}

const PROVIDERS = [
  { id: 'anthropic', name: 'Anthropic (Claude)', envKey: 'ANTHROPIC_API_KEY' },
  { id: 'openai', name: 'OpenAI (GPT-4)', envKey: 'OPENAI_API_KEY' },
  { id: 'gemini', name: 'Google Gemini', envKey: 'GEMINI_API_KEY' },
]

export function ProviderStep({ onNext, onSkip }: Props) {
  const token = useAuthStore((s) => s.token)
  const [selected, setSelected] = useState<string | null>(null)
  const [apiKey, setApiKey] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

  const handleSave = async () => {
    if (!token || !selected || !apiKey) return
    setSaving(true)
    setError('')
    try {
      await callEngine('providers.connect', { token, provider_id: selected, api_key: apiKey })
      setSaved(true)
    } catch (err) {
      setError(String(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <h2 className="text-lg font-semibold text-white text-center">Chọn AI Provider</h2>
      <p className="text-xs text-white/40 text-center">Nhập API key để kết nối. Có thể thêm sau trong Settings.</p>

      <div className="flex flex-col gap-2">
        {PROVIDERS.map((p) => (
          <button
            key={p.id}
            onClick={() => { setSelected(p.id); setSaved(false); setApiKey(''); setError('') }}
            className={`text-left px-4 py-3 rounded-lg border transition-colors ${
              selected === p.id
                ? 'bg-indigo-500/20 border-indigo-500/30 text-indigo-300'
                : 'bg-white/5 border-white/10 text-white/70 hover:bg-white/10'
            }`}
          >
            <div className="text-sm font-medium">{p.name}</div>
            <div className="text-xs text-white/30 mt-0.5">{p.envKey}</div>
          </button>
        ))}
      </div>

      {selected && !saved && (
        <div className="flex gap-2">
          <input
            type="password"
            placeholder="API Key"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            className="flex-1 bg-white/10 rounded-lg px-3 py-2 text-sm text-white outline-none focus:ring-2 ring-indigo-500/50 font-mono"
          />
          <button
            onClick={handleSave}
            disabled={!apiKey || saving}
            className="px-4 py-2 bg-indigo-500 hover:bg-indigo-400 text-white rounded-lg text-sm disabled:opacity-40 transition-colors shrink-0"
          >
            {saving ? '...' : 'Lưu'}
          </button>
        </div>
      )}

      {saved && (
        <p className="text-green-400 text-xs text-center">✓ Đã kết nối {selected}</p>
      )}

      {error && <p className="text-red-400 text-xs">{error}</p>}

      <div className="flex gap-2 mt-2">
        <button
          onClick={onSkip}
          className="flex-1 px-4 py-2 bg-white/5 hover:bg-white/10 text-white/40 rounded-lg text-sm transition-colors"
        >
          Bỏ qua
        </button>
        <button
          onClick={onNext}
          className="flex-1 px-4 py-2 bg-indigo-500 hover:bg-indigo-400 text-white rounded-lg text-sm font-medium transition-colors"
        >
          Tiếp tục
        </button>
      </div>
    </div>
  )
}
