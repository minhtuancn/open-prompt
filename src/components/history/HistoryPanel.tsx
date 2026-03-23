import { useEffect, useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

interface HistoryEntry {
  id: number
  query: string
  response: string
  provider: string
  model: string
  latency_ms: number
  status: string
  timestamp: string
}

export function HistoryPanel() {
  const token = useAuthStore((s) => s.token)
  const [entries, setEntries] = useState<HistoryEntry[]>([])
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)
  const [expanded, setExpanded] = useState<number | null>(null)

  const loadHistory = async () => {
    if (!token) return
    setLoading(true)
    try {
      const result = await callEngine<HistoryEntry[]>(
        search ? 'history.search' : 'history.list',
        search ? { token, search, limit: 30 } : { token, limit: 50, offset: 0 }
      )
      setEntries(result ?? [])
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { loadHistory() }, [token])

  const handleSearch = () => { loadHistory() }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') handleSearch()
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex gap-2">
        <input
          type="text"
          placeholder="Tìm trong lịch sử..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          onKeyDown={handleKeyDown}
          className="flex-1 bg-black/20 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50"
        />
        <button onClick={handleSearch}
          className="text-xs px-3 py-2 bg-indigo-500/80 hover:bg-indigo-500 text-white rounded-lg transition-colors">
          Tìm
        </button>
      </div>

      {loading && <p className="text-xs text-white/30">Đang tải...</p>}

      {entries.length === 0 && !loading && (
        <p className="text-xs text-white/30 text-center py-4">Chưa có lịch sử</p>
      )}

      {entries.map((entry) => (
        <div key={entry.id}
          className="bg-white/5 border border-white/10 rounded-xl p-3 cursor-pointer hover:bg-white/8 transition-colors"
          onClick={() => setExpanded(expanded === entry.id ? null : entry.id)}
        >
          <div className="flex items-center justify-between mb-1">
            <span className="text-xs text-white/40">{entry.timestamp}</span>
            <div className="flex items-center gap-2">
              <span className="text-xs text-indigo-400">{entry.provider}</span>
              <span className={`text-xs px-1.5 py-0.5 rounded ${entry.status === 'success' ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'}`}>
                {entry.status}
              </span>
            </div>
          </div>
          <p className="text-sm text-white/80 line-clamp-2">{entry.query}</p>
          {expanded === entry.id && entry.response && (
            <div className="mt-2 pt-2 border-t border-white/10">
              <p className="text-xs text-white/60 whitespace-pre-wrap max-h-40 overflow-y-auto">{entry.response}</p>
              <div className="mt-1 flex gap-3 text-xs text-white/30">
                <span>{entry.model}</span>
                <span>{entry.latency_ms}ms</span>
              </div>
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
