import { useEffect, useRef, useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'
import { GatewayForm } from './GatewayForm'
import { ModelPriorityList } from './ModelPriorityList'

interface Provider {
  id: string
  name: string
  auth_type: string
  connected: boolean
}

export function ProvidersTab() {
  const token = useAuthStore((s) => s.token)
  const [providers, setProviders] = useState<Provider[]>([])
  const [apiKeys, setApiKeys] = useState<Record<string, string>>({})
  const [saving, setSaving] = useState<Record<string, boolean>>({})
  const [saved, setSaved] = useState<Record<string, boolean>>({})
  const [error, setError] = useState<string>('')
  const timeoutRef = useRef<ReturnType<typeof setTimeout>>()

  useEffect(() => () => { if (timeoutRef.current) clearTimeout(timeoutRef.current) }, [])

  useEffect(() => {
    if (!token) return
    callEngine<Provider[]>('providers.list', { token })
      .then((list) => setProviders(list ?? []))
      .catch((e) => setError(String(e)))
  }, [token])

  const handleSaveKey = async (providerId: string) => {
    if (!token || !apiKeys[providerId]) return
    setSaving((p) => ({ ...p, [providerId]: true }))
    try {
      await callEngine('providers.connect', { token, provider_id: providerId, api_key: apiKeys[providerId]! })
      setSaved((p) => ({ ...p, [providerId]: true }))
      setProviders((prev) => prev.map((p) => p.id === providerId ? { ...p, connected: true } : p))
      if (timeoutRef.current) clearTimeout(timeoutRef.current)
      timeoutRef.current = setTimeout(() => setSaved((p) => ({ ...p, [providerId]: false })), 2000)
    } catch (e) {
      console.error(e)
    } finally {
      setSaving((p) => ({ ...p, [providerId]: false }))
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <p className="text-xs text-white/40">Nhập API key để kết nối AI provider.</p>
      {error && <p className="text-xs text-red-400 mb-2">{error}</p>}
      {providers.map((provider) => (
        <div key={provider.id} className="bg-white/5 border border-white/10 rounded-xl p-4">
          <div className="flex items-center justify-between mb-3">
            <div>
              <div className="text-sm font-medium text-white">{provider.name}</div>
              <div className="text-xs text-white/40">{provider.id}</div>
            </div>
            <span className={`text-xs px-2 py-0.5 rounded-full ${provider.connected ? 'bg-green-500/20 text-green-400' : 'bg-white/10 text-white/30'}`}>
              {provider.connected ? 'Đã kết nối' : 'Chưa kết nối'}
            </span>
          </div>
          {provider.auth_type === 'api_key' && (
            <div className="flex gap-2">
              <input
                type="password"
                placeholder={`${provider.name} API Key`}
                value={apiKeys[provider.id] ?? ''}
                onChange={(e) => setApiKeys((prev) => ({ ...prev, [provider.id]: e.target.value }))}
                className="flex-1 bg-black/20 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50 font-mono"
              />
              <button
                onClick={() => handleSaveKey(provider.id)}
                disabled={!apiKeys[provider.id] || saving[provider.id]}
                className="text-xs px-4 py-2 bg-indigo-500/80 hover:bg-indigo-500 text-white rounded-lg transition-colors disabled:opacity-40 shrink-0"
              >
                {saved[provider.id] ? '✓' : saving[provider.id] ? '...' : 'Lưu'}
              </button>
            </div>
          )}
        </div>
      ))}

      {/* Model Priority — drag-drop */}
      <div className="mt-4 pt-4 border-t border-white/10">
        <div className="text-sm font-medium text-white mb-3">Model Priority</div>
        <ModelPriorityList />
      </div>

      <div className="mt-4 pt-4 border-t border-white/10">
        <GatewayForm onAdded={() => {
          if (!token) return
          setError('')
          callEngine<Provider[]>('providers.list', { token })
            .then((list) => setProviders(list ?? []))
            .catch((e) => { console.error(e); setError('Không thể tải dữ liệu') })
        }} />
      </div>
    </div>
  )
}
