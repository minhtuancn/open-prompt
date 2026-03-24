import { useEffect, useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

interface ProviderInfo {
  id: string
  name: string
  connected: boolean
}

interface Props {
  query: string
  onSelect: (alias: string) => void
  visible: boolean
}

export function MentionHint({ query, onSelect, visible }: Props) {
  const token = useAuthStore((s) => s.token)
  const [providers, setProviders] = useState<ProviderInfo[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) return
    setError(null)
    callEngine<ProviderInfo[]>('providers.list', { token })
      .then((list) => setProviders(list ?? []))
      .catch((e) => { console.error(e); setError('Không thể tải dữ liệu') })
  }, [token])

  if (!visible || !query) return null

  const filtered = providers.filter(
    (p) => p.id.includes(query.toLowerCase()) || p.name.toLowerCase().includes(query.toLowerCase())
  )
  if (filtered.length === 0) return null

  return (
    <div className="absolute bottom-full left-5 mb-1 bg-surface border border-white/10 rounded-lg shadow-xl p-1 min-w-48 z-50">
      {error && <p className="text-red-400 text-sm px-3 py-2">{error}</p>}
      {filtered.map((p) => (
        <button
          key={p.id}
          onClick={() => onSelect(p.id)}
          className="w-full text-left px-3 py-1.5 rounded-md text-sm text-white/70 hover:bg-white/10 hover:text-white transition-colors"
        >
          <span className="text-indigo-400">@</span>{p.id}
          <span className="ml-2 text-xs text-white/30">{p.name}</span>
        </button>
      ))}
    </div>
  )
}
