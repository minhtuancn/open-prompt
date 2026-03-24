interface Props {
  onNext: () => void
}

export function WelcomeStep({ onNext }: Props) {
  return (
    <div className="flex flex-col items-center gap-6 py-6">
      <div className="text-4xl">✨</div>
      <h1 className="text-xl font-bold text-white">Chào mừng đến Open Prompt</h1>
      <p className="text-sm text-white/60 text-center max-w-sm">
        Desktop AI assistant hỗ trợ đa provider. Nhấn hotkey bất cứ lúc nào để gọi AI trợ giúp.
      </p>
      <button
        onClick={onNext}
        className="px-6 py-2 bg-indigo-500 hover:bg-indigo-400 text-white rounded-lg text-sm font-medium transition-colors"
      >
        Bắt đầu thiết lập
      </button>
    </div>
  )
}
