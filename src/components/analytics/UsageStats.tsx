import { useEffect, useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

type Period = '7d' | '30d' | '90d'

interface ProviderTotal {
  provider: string
  requests: number
  errors: number
  success_rate: number
}

interface DailySummary {
  date: string
  provider: string
  model: string
  requests: number
  errors: number
  avg_latency_ms: number
}

const PERIOD_LABELS: Record<Period, string> = { '7d': '7 ngày', '30d': '30 ngày', '90d': '90 ngày' }

export function UsageStats() {
  const token = useAuthStore((s) => s.token)
  const [period, setPeriod] = useState<Period>('7d')
  const [providers, setProviders] = useState<ProviderTotal[]>([])
  const [summary, setSummary] = useState<DailySummary[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!token) return
    setLoading(true)
    Promise.all([
      callEngine<{ providers: ProviderTotal[] }>('analytics.by_provider', { token, period }),
      callEngine<{ summary: DailySummary[] }>('analytics.summary', { token, period }),
    ])
      .then(([byProvider, bySummary]) => {
        setProviders(byProvider.providers ?? [])
        setSummary(bySummary.summary ?? [])
      })
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [period, token])

  const totalRequests = providers.reduce((sum, p) => sum + p.requests, 0)

  return (
    <div className="flex flex-col gap-4">
      <div className="flex gap-1">
        {(['7d', '30d', '90d'] as Period[]).map((p) => (
          <button
            key={p}
            onClick={() => setPeriod(p)}
            className={`text-xs px-3 py-1.5 rounded-lg transition-colors ${period === p ? 'bg-indigo-500/20 text-indigo-300' : 'text-white/40 hover:text-white/60'}`}
          >
            {PERIOD_LABELS[p]}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="text-white/30 text-sm text-center py-8">Đang tải...</div>
      ) : (
        <>
          <div className="bg-white/5 border border-white/10 rounded-xl p-4">
            <div className="text-xs text-white/40 mb-1">Tổng yêu cầu ({PERIOD_LABELS[period]})</div>
            <div className="text-3xl font-bold text-white">{totalRequests}</div>
          </div>

          {providers.length > 0 && (
            <div className="flex flex-col gap-2">
              <div className="text-xs text-white/40">Theo provider</div>
              {providers.map((p) => (
                <div key={p.provider} className="bg-white/5 border border-white/10 rounded-xl p-3 flex items-center justify-between">
                  <div>
                    <div className="text-sm font-medium text-white capitalize">{p.provider}</div>
                    <div className="text-xs text-white/40 mt-0.5">{p.errors} lỗi • {p.success_rate.toFixed(1)}% thành công</div>
                  </div>
                  <div className="text-right">
                    <div className="text-lg font-bold text-white">{p.requests}</div>
                    <div className="text-xs text-white/30">yêu cầu</div>
                  </div>
                </div>
              ))}
            </div>
          )}

          {summary.slice(0, 10).length > 0 && (
            <div className="flex flex-col gap-1">
              <div className="text-xs text-white/40 mb-1">Theo ngày</div>
              {summary.slice(0, 10).map((s, i) => (
                <div key={`${s.date}-${s.provider}-${s.model}-${i}`} className="flex items-center justify-between py-1.5 border-b border-white/5 last:border-0">
                  <div className="text-xs text-white/50 shrink-0">{s.date}</div>
                  <div className="text-xs text-white/30 mx-2 flex-1 truncate">{s.provider}/{s.model}</div>
                  <div className="text-xs text-white font-medium shrink-0">{s.requests} req</div>
                  <div className="text-xs text-white/30 ml-2 shrink-0">{s.avg_latency_ms}ms</div>
                </div>
              ))}
            </div>
          )}

          {totalRequests === 0 && (
            <div className="text-white/30 text-sm text-center py-6">Chưa có dữ liệu trong {PERIOD_LABELS[period]} qua.</div>
          )}
        </>
      )}
    </div>
  )
}
