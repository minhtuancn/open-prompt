import { useEffect, useRef, useState } from 'react'
import { invoke } from '@tauri-apps/api/core'
import { useOverlayStore } from '../../store/overlayStore'
import { FallbackDialog } from './FallbackDialog'
import { MarkdownRenderer } from './MarkdownRenderer'

export function ResponsePanel() {
  const { chunks, isStreaming, error, fallbackProviders, setFallbackProviders } = useOverlayStore()
  const text = chunks.join('')

  const [isInjecting, setIsInjecting] = useState(false)
  const [injectError, setInjectError] = useState<string | null>(null)
  const [injected, setInjected] = useState(false)
  const [injectedApp, setInjectedApp] = useState('')
  const [copied, setCopied] = useState(false)
  const injectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const copyTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    return () => {
      if (injectTimerRef.current) clearTimeout(injectTimerRef.current)
      if (copyTimerRef.current) clearTimeout(copyTimerRef.current)
    }
  }, [])

  if (!text && !isStreaming && !error) return null

  const handleInject = async () => {
    if (!text || isInjecting) return
    setIsInjecting(true)
    setInjectError(null)
    setInjected(false)
    try {
      const appName = await invoke<string>('inject_text', { text })
      setInjected(true)
      setInjectedApp(appName || '')
      injectTimerRef.current = setTimeout(() => { setInjected(false); setInjectedApp('') }, 3000)
    } catch (err) {
      setInjectError(err as string)
    } finally {
      setIsInjecting(false)
    }
  }

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      if (copyTimerRef.current) clearTimeout(copyTimerRef.current)
      copyTimerRef.current = setTimeout(() => setCopied(false), 2000)
    } catch {
      // Fallback cho khi clipboard API không available
      const textarea = document.createElement('textarea')
      textarea.value = text
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand('copy')
      document.body.removeChild(textarea)
      setCopied(true)
      if (copyTimerRef.current) clearTimeout(copyTimerRef.current)
      copyTimerRef.current = setTimeout(() => setCopied(false), 2000)
    }
  }

  const handleFallbackRetry = (provider: string) => {
    window.dispatchEvent(new CustomEvent('fallback-retry', { detail: { provider } }))
  }

  return (
    <div className="px-5 pb-4 max-h-80 overflow-y-auto">
      <div className="border-t border-white/10 pt-3">
        {error ? (
          <>
            <p className="text-red-400 text-sm">{error}</p>
            {fallbackProviders.length > 0 && (
              <FallbackDialog
                errorMessage={error}
                providers={fallbackProviders}
                onRetry={handleFallbackRetry}
                onCancel={() => setFallbackProviders([])}
              />
            )}
          </>
        ) : (
          <>
            {isStreaming ? (
              <p className="text-white/90 text-sm leading-relaxed whitespace-pre-wrap">
                {text}<span className="animate-pulse">▌</span>
              </p>
            ) : (
              <MarkdownRenderer text={text} />
            )}

            {text && !isStreaming && (
              <div className="mt-3 flex items-center gap-2">
                <button
                  onClick={handleInject}
                  disabled={isInjecting}
                  className={`
                    flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium
                    transition-all duration-150
                    ${injected
                      ? 'bg-green-500/20 text-green-400 border border-green-500/30'
                      : 'bg-white/10 text-white/70 border border-white/20 hover:bg-white/20 hover:text-white'
                    }
                    disabled:opacity-50 disabled:cursor-not-allowed
                  `}
                  title="Chèn text vào ứng dụng đang focus"
                >
                  {isInjecting ? (
                    <><span className="animate-spin">⟳</span><span>Đang chèn...</span></>
                  ) : injected ? (
                    <><span>✓</span><span>Đã chèn{injectedApp ? ` → ${injectedApp}` : ''}</span></>
                  ) : (
                    <span>Insert ↵</span>
                  )}
                </button>

                <button
                  onClick={handleCopy}
                  className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all duration-150 ${
                    copied
                      ? 'bg-green-500/20 text-green-400 border border-green-500/30'
                      : 'bg-white/5 text-white/40 border border-white/10 hover:bg-white/10 hover:text-white/70'
                  }`}
                >
                  {copied ? '✓ Copied' : 'Copy'}
                </button>

                {injectError && (
                  <span className="text-red-400 text-xs">{injectError}</span>
                )}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}
