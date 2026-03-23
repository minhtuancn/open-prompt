import { useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

const PRESETS = [
  { name: 'ollama', displayName: 'Ollama (Local)', baseURL: 'http://localhost:11434/v1', defaultModel: 'llama3.2' },
  { name: 'litellm', displayName: 'LiteLLM', baseURL: 'http://localhost:4000/v1', defaultModel: 'gpt-4o' },
  { name: 'openrouter', displayName: 'OpenRouter', baseURL: 'https://openrouter.ai/api/v1', defaultModel: 'openai/gpt-4o' },
  { name: 'vllm', displayName: 'vLLM', baseURL: 'http://localhost:8000/v1', defaultModel: '' },
]

interface Props {
  onAdded?: () => void
}

export function GatewayForm({ onAdded }: Props) {
  const token = useAuthStore((s) => s.token)
  const [name, setName] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [baseURL, setBaseURL] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [defaultModel, setDefaultModel] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)

  const applyPreset = (preset: typeof PRESETS[0]) => {
    setName(preset.name)
    setDisplayName(preset.displayName)
    setBaseURL(preset.baseURL)
    setDefaultModel(preset.defaultModel)
  }

  const handleSubmit = async () => {
    if (!token || !name || !baseURL) return
    setSaving(true)
    setError('')
    try {
      await callEngine('providers.add_gateway', {
        token, name, display_name: displayName || name, base_url: baseURL,
        api_key: apiKey, default_model: defaultModel,
      })
      setSuccess(true)
      setTimeout(() => setSuccess(false), 2000)
      setName(''); setDisplayName(''); setBaseURL(''); setApiKey(''); setDefaultModel('')
      onAdded?.()
    } catch (e) {
      setError(String(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="bg-white/5 border border-white/10 rounded-xl p-4">
      <div className="text-sm font-medium text-white mb-3">Thêm Gateway</div>

      <div className="flex flex-wrap gap-1.5 mb-3">
        {PRESETS.map((p) => (
          <button key={p.name} onClick={() => applyPreset(p)}
            className="text-xs px-2 py-1 bg-white/5 hover:bg-indigo-500/20 text-white/50 hover:text-white rounded-md border border-white/10 transition-colors">
            {p.displayName}
          </button>
        ))}
      </div>

      <div className="flex flex-col gap-2">
        <input placeholder="Tên (vd: my-ollama)" value={name} onChange={(e) => setName(e.target.value)}
          className="bg-black/20 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50" />
        <input placeholder="Base URL (vd: http://localhost:11434/v1)" value={baseURL} onChange={(e) => setBaseURL(e.target.value)}
          className="bg-black/20 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50 font-mono" />
        <div className="flex gap-2">
          <input placeholder="API Key (tùy chọn)" type="password" value={apiKey} onChange={(e) => setApiKey(e.target.value)}
            className="flex-1 bg-black/20 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50 font-mono" />
          <input placeholder="Model mặc định" value={defaultModel} onChange={(e) => setDefaultModel(e.target.value)}
            className="flex-1 bg-black/20 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50" />
        </div>
        <div className="flex items-center gap-2">
          <button onClick={handleSubmit} disabled={!name || !baseURL || saving}
            className="text-xs px-4 py-2 bg-indigo-500/80 hover:bg-indigo-500 text-white rounded-lg transition-colors disabled:opacity-40">
            {success ? '✓ Đã thêm' : saving ? '...' : 'Thêm Gateway'}
          </button>
          {error && <span className="text-xs text-red-400">{error}</span>}
        </div>
      </div>
    </div>
  )
}
