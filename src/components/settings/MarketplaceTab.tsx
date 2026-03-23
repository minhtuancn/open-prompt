import { useEffect, useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

interface SharedPrompt {
  id: number
  title: string
  content: string
  description: string
  category: string
  tags: string
  downloads: number
}

export function MarketplaceTab() {
  const token = useAuthStore((s) => s.token)
  const [prompts, setPrompts] = useState<SharedPrompt[]>([])
  const [search, setSearch] = useState('')
  const [installed, setInstalled] = useState<Set<number>>(new Set())
  const [loading, setLoading] = useState(true)

  const fetchPrompts = async (query?: string) => {
    if (!token) return
    setLoading(true)
    try {
      const method = query ? 'marketplace.search' : 'marketplace.list'
      const params = query ? { token, query, limit: 50 } : { token, limit: 50, offset: 0 }
      const result = await callEngine<SharedPrompt[]>(method, params)
      setPrompts(result ?? [])
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchPrompts() }, [token])

  const handleSearch = () => {
    if (search.trim()) {
      fetchPrompts(search.trim())
    } else {
      fetchPrompts()
    }
  }

  const handleInstall = async (id: number) => {
    if (!token) return
    try {
      await callEngine('marketplace.install', { token, id })
      setInstalled((prev) => new Set(prev).add(id))
    } catch (e) {
      console.error(e)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <p className="text-xs text-white/40">Duyệt và cài đặt prompts từ cộng đồng.</p>

      {/* Search bar */}
      <div className="flex gap-2">
        <input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          placeholder="Tìm kiếm prompts..."
          className="flex-1 bg-black/20 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50"
        />
        <button
          onClick={handleSearch}
          className="px-4 py-2 bg-indigo-500/80 hover:bg-indigo-500 text-white rounded-lg text-xs transition-colors shrink-0"
        >
          Tìm
        </button>
      </div>

      {/* Prompt list */}
      {loading ? (
        <p className="text-xs text-white/30 text-center py-4">Đang tải...</p>
      ) : prompts.length === 0 ? (
        <p className="text-xs text-white/30 text-center py-4">Chưa có prompt nào. Hãy publish prompt đầu tiên!</p>
      ) : (
        <div className="flex flex-col gap-2">
          {prompts.map((p) => (
            <div key={p.id} className="bg-white/5 border border-white/10 rounded-xl p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium text-white">{p.title}</div>
                  {p.description && (
                    <div className="text-xs text-white/40 mt-1">{p.description}</div>
                  )}
                  <div className="flex items-center gap-3 mt-2">
                    {p.category && (
                      <span className="text-xs bg-indigo-500/10 text-indigo-300 px-2 py-0.5 rounded">{p.category}</span>
                    )}
                    <span className="text-xs text-white/30">{p.downloads} downloads</span>
                  </div>
                </div>
                <button
                  onClick={() => handleInstall(p.id)}
                  disabled={installed.has(p.id)}
                  className={`shrink-0 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                    installed.has(p.id)
                      ? 'bg-green-500/20 text-green-400 border border-green-500/30'
                      : 'bg-indigo-500/80 hover:bg-indigo-500 text-white'
                  }`}
                >
                  {installed.has(p.id) ? '✓ Đã cài' : 'Cài đặt'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
