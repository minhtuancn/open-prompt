import { useSettingsStore } from '../../store/settingsStore'

const LOCALES = [
  { value: 'vi' as const, label: 'Tiếng Việt', flag: '🇻🇳' },
  { value: 'en' as const, label: 'English', flag: '🇬🇧' },
]

export function LanguageTab() {
  const { locale, setLocale } = useSettingsStore()

  return (
    <div className="flex flex-col gap-2">
      <p className="text-xs text-white/40 mb-2">Chọn ngôn ngữ hiển thị của ứng dụng.</p>
      {LOCALES.map((l) => (
        <button
          key={l.value}
          onClick={() => setLocale(l.value)}
          className={`flex items-center gap-3 px-4 py-3 rounded-xl border text-left transition-colors ${locale === l.value ? 'border-indigo-500 bg-indigo-500/15 text-white' : 'border-white/10 bg-white/5 text-white/60 hover:bg-white/10'}`}
        >
          <span className="text-xl">{l.flag}</span>
          <span className="text-sm font-medium">{l.label}</span>
          {locale === l.value && <span className="ml-auto text-indigo-400 text-xs">✓ Đang dùng</span>}
        </button>
      ))}
    </div>
  )
}
