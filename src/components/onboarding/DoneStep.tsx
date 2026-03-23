interface Props {
  onComplete: () => void
}

export function DoneStep({ onComplete }: Props) {
  return (
    <div className="flex flex-col items-center gap-6 py-6">
      <div className="text-4xl">🚀</div>
      <h2 className="text-lg font-semibold text-white">Sẵn sàng!</h2>
      <p className="text-sm text-white/60 text-center max-w-sm">
        Nhấn phím tắt đã chọn để mở overlay và bắt đầu dùng AI assistant.
      </p>
      <div className="bg-white/5 border border-white/10 rounded-lg px-4 py-3 text-xs text-white/50">
        <div><kbd className="text-indigo-400 font-mono">@claude</kbd> — dùng Claude</div>
        <div><kbd className="text-indigo-400 font-mono">@gpt4</kbd> — dùng GPT-4</div>
        <div><kbd className="text-indigo-400 font-mono">Ctrl+M</kbd> — chọn model</div>
        <div><kbd className="text-indigo-400 font-mono">/</kbd> — slash commands</div>
      </div>
      <button
        onClick={onComplete}
        className="px-6 py-2 bg-indigo-500 hover:bg-indigo-400 text-white rounded-lg text-sm font-medium transition-colors"
      >
        Bắt đầu dùng
      </button>
    </div>
  )
}
