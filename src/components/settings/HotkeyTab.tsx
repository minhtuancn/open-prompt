import { useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

const HOTKEY_OPTIONS = [
  { label: 'Ctrl + Space', value: 'ctrl+space' },
  { label: 'Ctrl + Shift + Space', value: 'ctrl+shift+space' },
  { label: 'Alt + Space', value: 'alt+space' },
  { label: 'Ctrl + /', value: 'ctrl+/' },
  { label: 'Ctrl + J', value: 'ctrl+j' },
  { label: 'Super + Space', value: 'super+space' },
]

export function HotkeyTab() {
  const token = useAuthStore((s) => s.token)
  const [selected, setSelected] = useState('ctrl+space')
  const [saved, setSaved] = useState(false)

  const handleSave = async () => {
    if (!token) return
    try {
      await callEngine('settings.set', { token, key: 'hotkey', value: selected })
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch (e) {
      console.error(e)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <p className="text-xs text-white/40">Chọn phím tắt để mở overlay. Cần restart app để áp dụng.</p>

      <div className="bg-white/5 border border-white/10 rounded-xl p-4">
        <div className="text-xs text-white/40 mb-3">Hotkey hiện tại</div>
        <div className="grid grid-cols-2 gap-2">
          {HOTKEY_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              onClick={() => setSelected(opt.value)}
              className={`flex items-center justify-center gap-1 px-3 py-2 rounded-lg text-xs font-mono transition-colors ${
                selected === opt.value
                  ? 'bg-indigo-500/20 text-indigo-300 border border-indigo-500/30'
                  : 'bg-white/5 text-white/50 border border-white/10 hover:bg-white/10'
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>

        <div className="mt-3 flex items-center gap-2">
          <button
            onClick={handleSave}
            className="text-xs px-4 py-2 bg-indigo-500/80 hover:bg-indigo-500 text-white rounded-lg transition-colors"
          >
            {saved ? '✓ Đã lưu' : 'Lưu hotkey'}
          </button>
          <span className="text-xs text-white/30">Cần restart app</span>
        </div>
      </div>

      <div className="bg-white/5 border border-white/10 rounded-xl p-4">
        <div className="text-xs text-white/40 mb-2">Phím tắt trong overlay</div>
        <div className="flex flex-col gap-1.5 text-xs text-white/50">
          <div><kbd className="text-indigo-400 font-mono">Ctrl+M</kbd> — Chọn model/provider</div>
          <div><kbd className="text-indigo-400 font-mono">@</kbd> — Mention provider</div>
          <div><kbd className="text-indigo-400 font-mono">/</kbd> — Slash commands</div>
          <div><kbd className="text-indigo-400 font-mono">Enter</kbd> — Gửi query</div>
          <div><kbd className="text-indigo-400 font-mono">Shift+Enter</kbd> — Xuống dòng</div>
          <div><kbd className="text-indigo-400 font-mono">Escape</kbd> — Đóng overlay</div>
        </div>
      </div>
    </div>
  )
}
