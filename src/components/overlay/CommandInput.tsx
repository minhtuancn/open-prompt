import { useOverlayStore } from '../../store/overlayStore'

interface Props {
  onSubmit: (input: string) => void
}

export function CommandInput({ onSubmit }: Props) {
  const { input, setInput, isStreaming } = useOverlayStore()

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      if (input.trim() && !isStreaming) {
        onSubmit(input.trim())
      }
    }
    if (e.key === 'Escape') {
      window.close()
    }
  }

  return (
    <div className="relative">
      <textarea
        autoFocus
        rows={1}
        className="w-full bg-transparent text-white text-lg placeholder-white/40 outline-none resize-none px-5 py-4 leading-relaxed"
        placeholder="Hỏi AI... (Enter để gửi, Shift+Enter xuống dòng)"
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={handleKeyDown}
        disabled={isStreaming}
      />
    </div>
  )
}
