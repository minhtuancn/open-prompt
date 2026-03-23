import { useEffect, useState } from 'react'
import { callEngine, streamQuery } from './hooks/useEngine'
import { useAuthStore } from './store/authStore'
import { useOverlayStore } from './store/overlayStore'
import { CreateAccount } from './components/onboarding/CreateAccount'
import { LoginScreen } from './components/auth/LoginScreen'
import { CommandInput } from './components/overlay/CommandInput'
import { ResponsePanel } from './components/overlay/ResponsePanel'
import { ApiKeySetup } from './components/settings/ApiKeySetup'
import { SettingsLayout } from './components/settings/SettingsLayout'
import './styles/globals.css'

type AppState = 'loading' | 'first-run' | 'login' | 'api-setup' | 'overlay' | 'settings'

export default function App() {
  const [state, setState] = useState<AppState>('loading')
  const { token } = useAuthStore()
  const { reset, appendChunk, setStreaming, setError } = useOverlayStore()

  useEffect(() => {
    async function init() {
      try {
        const result = await callEngine<{ is_first_run: boolean }>('auth.is_first_run', {})
        if (result.is_first_run) {
          setState('first-run')
        } else if (!token) {
          setState('login')
        } else {
          setState('overlay')
        }
      } catch {
        setState('overlay')
      }
    }
    init()
  }, [token])

  const handleQuery = async (input: string) => {
    if (!token) return
    reset()
    setStreaming(true)
    await streamQuery(
      { token, input },
      (chunk) => appendChunk(chunk),
      () => setStreaming(false),
      (err) => { setError(err); setStreaming(false) }
    )
  }

  if (state === 'loading') {
    return (
      <div className="flex items-center justify-center h-screen bg-surface">
        <div className="text-white/40">Đang khởi động...</div>
      </div>
    )
  }

  if (state === 'first-run') {
    return <CreateAccount onDone={() => setState('api-setup')} />
  }

  if (state === 'login') {
    return <LoginScreen onDone={() => setState('overlay')} />
  }

  if (state === 'api-setup') {
    return <ApiKeySetup onDone={() => setState('overlay')} />
  }

  if (state === 'settings') {
    return (
      <div className="bg-surface/95 backdrop-blur-xl rounded-2xl border border-white/10 shadow-2xl overflow-hidden">
        <SettingsLayout onClose={() => setState('overlay')} />
      </div>
    )
  }

  return (
    <div className="bg-surface/95 backdrop-blur-xl rounded-2xl border border-white/10 shadow-2xl overflow-hidden min-h-16">
      <div className="flex items-start">
        <div className="flex-1 min-w-0">
          <CommandInput onSubmit={handleQuery} />
        </div>
        <button
          onClick={() => setState('settings')}
          title="Cài đặt"
          className="p-3 mt-2 mr-2 text-white/25 hover:text-white/60 transition-colors text-base shrink-0"
        >
          ⚙
        </button>
      </div>
      <ResponsePanel />
    </div>
  )
}
