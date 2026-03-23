import { useState } from 'react'
import { invoke } from '@tauri-apps/api/core'
import { useOverlayStore } from '../../store/overlayStore'

export function ResponsePanel() {
  const { chunks, isStreaming, error } = useOverlayStore()
  const text = chunks.join('')

  const [isInjecting, setIsInjecting] = useState(false)
  const [injectError, setInjectError] = useState<string | null>(null)
  const [injected, setInjected] = useState(false)

  if (!text && !isStreaming && !error) return null

  const handleInject = async () => {
    if (!text || isInjecting) return
    setIsInjecting(true)
    setInjectError(null)
    setInjected(false)
    try {
      await invoke('inject_text', { text })
      setInjected(true)
      setTimeout(() => setInjected(false), 2000)
    } catch (err) {
      setInjectError(err as string)
    } finally {
      setIsInjecting(false)
    }
  }

  return (
    <div className="px-5 pb-4 max-h-80 overflow-y-auto">
      <div className="border-t border-white/10 pt-3">
        {error ? (
          <p className="text-red-400 text-sm">{error}</p>
        ) : (
          <>
            <p className="text-white/90 text-sm leading-relaxed whitespace-pre-wrap">
              {text}
              {isStreaming && <span className="animate-pulse">▌</span>}
            </p>

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
                    <>
                      <span className="animate-spin">⟳</span>
                      <span>Đang chèn...</span>
                    </>
                  ) : injected ? (
                    <>
                      <span>✓</span>
                      <span>Đã chèn</span>
                    </>
                  ) : (
                    <span>Insert ↵</span>
                  )}
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
