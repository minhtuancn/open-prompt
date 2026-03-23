import { useSettingsStore } from '../../store/settingsStore'

const FONT_SIZES = [
  { value: 'sm' as const, label: 'Nhỏ (13px)' },
  { value: 'base' as const, label: 'Vừa (14px)' },
  { value: 'lg' as const, label: 'Lớn (16px)' },
]

export function AppearanceTab() {
  const { fontSize, setFontSize } = useSettingsStore()

  return (
    <div className="flex flex-col gap-4">
      <div>
        <label className="text-xs text-white/50 mb-2 block">Cỡ chữ</label>
        <div className="flex gap-2">
          {FONT_SIZES.map((size) => (
            <button
              key={size.value}
              onClick={() => setFontSize(size.value)}
              className={`flex-1 text-xs py-2 px-3 rounded-lg border transition-colors ${fontSize === size.value ? 'border-indigo-500 bg-indigo-500/20 text-indigo-300' : 'border-white/10 bg-white/5 text-white/50 hover:bg-white/10'}`}
            >
              {size.label}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
