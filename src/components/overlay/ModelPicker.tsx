import { useEffect, useState, useMemo } from 'react'
import { useAuthStore } from '../../store/authStore'
import { useProviderStore } from '../../store/providerStore'

interface Props {
  onSelect: (providerName: string) => void
  onClose: () => void
}

export function ModelPicker({ onSelect, onClose }: Props) {
  const token = useAuthStore((s) => s.token)
  const allProviders = useProviderStore((s) => s.providers)
  const error = useProviderStore((s) => s.error)
  const fetchProviders = useProviderStore((s) => s.fetch)
  const [selectedIdx, setSelectedIdx] = useState(0)

  useEffect(() => {
    if (token) fetchProviders(token)
  }, [token, fetchProviders])

  const providers = useMemo(() => {
    const connected = allProviders.filter((p) => p.connected)
    return connected.length > 0 ? connected : allProviders
  }, [allProviders])

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { onClose(); return }
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setSelectedIdx((i) => Math.min(i + 1, providers.length - 1))
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        setSelectedIdx((i) => Math.max(i - 1, 0))
      }
      if (e.key === 'Enter' && providers.length > 0) {
        e.preventDefault()
        onSelect(providers[selectedIdx].id)
        onClose()
      }
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [providers, selectedIdx, onSelect, onClose])

  if (providers.length === 0) return null

  return (
    <div className="absolute top-0 left-0 right-0 z-50 bg-surface/98 backdrop-blur-xl border border-white/10 rounded-xl shadow-2xl p-2" role="listbox" aria-label="Chọn AI provider">
      <div className="flex items-center justify-between px-3 py-1.5 mb-1">
        <span className="text-xs text-white/50 font-medium">Chọn provider</span>
        <span className="text-xs text-white/30">ESC để đóng</span>
      </div>
      {error && <p className="text-red-400 text-sm px-3 py-2">{error}</p>}
      {providers.map((p, i) => (
        <button
          key={p.id}
          onClick={() => { onSelect(p.id); onClose() }}
          aria-label={p.name}
          className={`w-full text-left px-3 py-2 rounded-lg text-sm transition-colors ${
            i === selectedIdx
              ? 'bg-indigo-500/20 text-white'
              : 'text-white/70 hover:bg-white/5'
          }`}
        >
          <span className="font-medium">{p.name}</span>
          {p.connected && <span className="ml-2 text-xs text-green-400/70">●</span>}
        </button>
      ))}
    </div>
  )
}
