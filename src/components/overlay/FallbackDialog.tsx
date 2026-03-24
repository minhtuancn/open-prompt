interface Props {
  errorMessage: string
  providers: string[]
  onRetry: (provider: string) => void
  onCancel: () => void
}

export function FallbackDialog({ errorMessage, providers, onRetry, onCancel }: Props) {
  return (
    <div className="mt-3 bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-3">
      <p className="text-yellow-400 text-xs mb-2">⚠ {errorMessage}</p>
      <p className="text-white/50 text-xs mb-2">Thử lại với:</p>
      <div className="flex flex-wrap gap-2">
        {providers.map((name) => (
          <button
            key={name}
            onClick={() => onRetry(name)}
            aria-label={"Thử lại với " + name}
            className="px-3 py-1 bg-white/10 hover:bg-indigo-500/30 text-white/80 hover:text-white text-xs rounded-md border border-white/10 hover:border-indigo-500/30 transition-colors"
          >
            {name}
          </button>
        ))}
        <button
          onClick={onCancel}
          aria-label="Huỷ"
          className="px-3 py-1 text-white/30 hover:text-white/60 text-xs transition-colors"
        >
          Hủy
        </button>
      </div>
    </div>
  )
}
