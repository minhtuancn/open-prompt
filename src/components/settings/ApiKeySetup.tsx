import { useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

interface Props {
  onDone: () => void
}

export function ApiKeySetup({ onDone }: Props) {
  const [apiKey, setApiKey] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const { token } = useAuthStore()

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!apiKey.startsWith('sk-ant-')) {
      setError('Claude API key phải bắt đầu bằng sk-ant-')
      return
    }
    setLoading(true)
    try {
      await callEngine('settings.set', {
        token,
        key: 'anthropic_api_key',
        value: apiKey,
      })
      onDone()
    } catch (err: unknown) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col items-center justify-center h-screen bg-surface text-white">
      <h2 className="text-xl font-bold mb-2">Cấu hình AI Provider</h2>
      <p className="text-white/60 text-sm mb-6 text-center px-8">
        Nhập Anthropic API key để bắt đầu dùng Claude
      </p>
      <form onSubmit={handleSave} className="flex flex-col gap-3 w-80">
        <input
          autoFocus
          type="password"
          className="bg-white/10 rounded-lg px-4 py-2 outline-none focus:ring-2 ring-accent font-mono text-sm"
          placeholder="sk-ant-api03-..."
          value={apiKey}
          onChange={(e) => setApiKey(e.target.value)}
          required
        />
        {error && <p className="text-red-400 text-sm">{error}</p>}
        <button
          type="submit"
          disabled={loading}
          className="bg-accent rounded-lg py-2 font-semibold disabled:opacity-50 hover:bg-indigo-500 transition"
        >
          {loading ? 'Đang lưu...' : 'Lưu và tiếp tục'}
        </button>
        <button
          type="button"
          onClick={onDone}
          className="text-white/40 text-sm hover:text-white/60"
        >
          Bỏ qua (cấu hình sau)
        </button>
      </form>
    </div>
  )
}
