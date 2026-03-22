import { useOverlayStore } from '../../store/overlayStore'

export function ResponsePanel() {
  const { chunks, isStreaming, error } = useOverlayStore()
  const text = chunks.join('')

  if (!text && !isStreaming && !error) return null

  return (
    <div className="px-5 pb-4 max-h-80 overflow-y-auto">
      <div className="border-t border-white/10 pt-3">
        {error ? (
          <p className="text-red-400 text-sm">{error}</p>
        ) : (
          <p className="text-white/90 text-sm leading-relaxed whitespace-pre-wrap">
            {text}
            {isStreaming && <span className="animate-pulse">▌</span>}
          </p>
        )}
      </div>
    </div>
  )
}
