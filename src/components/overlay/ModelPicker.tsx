import { useEffect, useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

interface ProviderInfo {
  id: string
  name: string
  connected: boolean
}

interface Props {
  onSelect: (providerName: string) => void
  onClose: () => void
}

export function ModelPicker({ onSelect, onClose }: Props) {
  const token = useAuthStore((s) => s.token)
  const [providers, setProviders] = useState<ProviderInfo[]>([])
  const [selectedIdx, setSelectedIdx] = useState(0)

  useEffect(() => {
    if (!token) return
    callEngine<ProviderInfo[]>('providers.list', { token })
      .then((list) => {
        const connected = (list ?? []).filter((p) => p.connected)
        setProviders(connected.length > 0 ? connected : list ?? [])
      })
      .catch(console.error)
  }, [token])

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
    <div className="absolute top-0 left-0 right-0 z-50 bg-surface/98 backdrop-blur-xl border border-white/10 rounded-xl shadow-2xl p-2">
      <div className="flex items-center justify-between px-3 py-1.5 mb-1">
        <span className="text-xs text-white/50 font-medium">Chọn provider</span>
        <span className="text-xs text-white/30">ESC để đóng</span>
      </div>
      {providers.map((p, i) => (
        <button
          key={p.id}
          onClick={() => { onSelect(p.id); onClose() }}
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
