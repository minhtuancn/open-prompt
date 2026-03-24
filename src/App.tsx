import { useEffect, useState } from 'react'
import { callEngine, streamQuery } from './hooks/useEngine'
import { useAuthStore } from './store/authStore'
import { useOverlayStore } from './store/overlayStore'
import { useSettingsStore } from './store/settingsStore'
import { OnboardingWizard } from './components/onboarding/OnboardingWizard'
import { LoginScreen } from './components/auth/LoginScreen'
import { CommandInput } from './components/overlay/CommandInput'
import { ResponsePanel } from './components/overlay/ResponsePanel'
import { SettingsLayout } from './components/settings/SettingsLayout'
import './styles/globals.css'

type AppState = 'loading' | 'onboarding' | 'login' | 'overlay' | 'settings'

const FONT_SIZE_CLASS: Record<string, string> = {
  sm: 'text-sm',
  base: 'text-base',
  lg: 'text-lg',
}

export default function App() {
  const [state, setState] = useState<AppState>('loading')
  const { token } = useAuthStore()
  const { reset, appendChunk, setStreaming, setError, setFallbackProviders, setLastQuery, activeProvider } = useOverlayStore((s) => ({
    reset: s.reset,
    appendChunk: s.appendChunk,
    setStreaming: s.setStreaming,
    setError: s.setError,
    setFallbackProviders: s.setFallbackProviders,
    setLastQuery: s.setLastQuery,
    activeProvider: s.activeProvider,
  }))
  const fontSize = useSettingsStore((s) => s.fontSize)
  const fontSizeClass = FONT_SIZE_CLASS[fontSize]

  useEffect(() => {
    async function init() {
      try {
        const result = await callEngine<{ is_first_run: boolean }>('auth.is_first_run', {})
        if (result.is_first_run) {
          setState('onboarding')
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

  // Fallback retry: lắng nghe event từ FallbackDialog
  useEffect(() => {
    const handler = (e: Event) => {
      const { provider } = (e as CustomEvent).detail
      const lastQuery = useOverlayStore.getState().lastQuery
      if (!token || !lastQuery) return
      const store = useOverlayStore.getState()
      store.reset()
      store.setStreaming(true)
      store.setLastQuery(lastQuery)
      streamQuery(
        { token, input: lastQuery, provider },
        (chunk) => useOverlayStore.getState().appendChunk(chunk),
        () => useOverlayStore.getState().setStreaming(false),
        (err, fallback) => {
          useOverlayStore.getState().setError(err)
          useOverlayStore.getState().setStreaming(false)
          if (fallback && fallback.length > 0) useOverlayStore.getState().setFallbackProviders(fallback)
        }
      )
    }
    window.addEventListener('fallback-retry', handler)
    return () => window.removeEventListener('fallback-retry', handler)
  }, [token])

  const handleQuery = async (input: string) => {
    if (!token) return
    reset()
    setStreaming(true)
    setLastQuery(input)
    await streamQuery(
      { token, input, provider: activeProvider || undefined },
      (chunk) => appendChunk(chunk),
      () => setStreaming(false),
      (err, fallback) => {
        setError(err)
        setStreaming(false)
        if (fallback && fallback.length > 0) setFallbackProviders(fallback)
      }
    )
  }

  if (state === 'loading') {
    return (
      <div className="flex items-center justify-center h-screen bg-surface">
        <div className="text-white/40">Đang khởi động...</div>
      </div>
    )
  }

  if (state === 'onboarding') {
    return <OnboardingWizard onComplete={() => setState('overlay')} />
  }

  if (state === 'login') {
    return <LoginScreen onDone={() => setState('overlay')} />
  }

  if (state === 'settings') {
    return (
      <div className={`bg-surface/95 backdrop-blur-xl rounded-2xl border border-white/10 shadow-2xl overflow-hidden ${fontSizeClass}`}>
        <SettingsLayout onClose={() => setState('overlay')} />
      </div>
    )
  }

  return (
    <div className={`bg-surface/95 backdrop-blur-xl rounded-2xl border border-white/10 shadow-2xl overflow-hidden min-h-16 ${fontSizeClass}`}>
      <div className="flex items-start">
        <div className="flex-1 min-w-0">
          <CommandInput onSubmit={handleQuery} />
        </div>
        <button
          onClick={() => setState('settings')}
          title="Cài đặt"
          aria-label="Cài đặt"
          className="p-3 mt-2 mr-2 text-white/25 hover:text-white/60 transition-colors text-base shrink-0"
        >
          ⚙
        </button>
      </div>
      <ResponsePanel />
    </div>
  )
}
