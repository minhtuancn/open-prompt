import { useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

interface Props {
  onNext: () => void
}

const HOTKEY_OPTIONS = [
  { label: 'Ctrl + Space', value: 'ctrl+space' },
  { label: 'Ctrl + Shift + Space', value: 'ctrl+shift+space' },
  { label: 'Alt + Space', value: 'alt+space' },
  { label: 'Ctrl + /', value: 'ctrl+/' },
  { label: 'Ctrl + J', value: 'ctrl+j' },
  { label: 'Super + Space', value: 'super+space' },
]

export function HotkeyStep({ onNext }: Props) {
  const token = useAuthStore((s) => s.token)
  const [selected, setSelected] = useState('ctrl+space')

  const handleNext = async () => {
    if (token) {
      try {
        await callEngine('settings.set', { token, key: 'hotkey', value: selected })
      } catch {
        // Không block wizard nếu lưu thất bại
      }
    }
    onNext()
  }

  return (
    <div className="flex flex-col gap-4">
      <h2 className="text-lg font-semibold text-white text-center">Chọn phím tắt</h2>
      <p className="text-xs text-white/40 text-center">Nhấn phím tắt này để mở overlay bất cứ lúc nào</p>

      <div className="grid grid-cols-2 gap-2">
        {HOTKEY_OPTIONS.map((opt) => (
          <button
            key={opt.value}
            onClick={() => setSelected(opt.value)}
            className={`flex items-center justify-center px-3 py-2.5 rounded-lg text-sm font-mono transition-colors ${
              selected === opt.value
                ? 'bg-indigo-500/20 text-indigo-300 border border-indigo-500/30'
                : 'bg-white/5 text-white/50 border border-white/10 hover:bg-white/10'
            }`}
          >
            {opt.label}
          </button>
        ))}
      </div>

      <button
        onClick={handleNext}
        className="mt-2 px-6 py-2 bg-indigo-500 hover:bg-indigo-400 text-white rounded-lg text-sm font-medium transition-colors"
      >
        Tiếp tục
      </button>
    </div>
  )
}
