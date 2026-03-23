import vi from './locales/vi.json'
import en from './locales/en.json'
import fr from './locales/fr.json'
import zhCN from './locales/zh-CN.json'
import th from './locales/th.json'
import lo from './locales/lo.json'
import ru from './locales/ru.json'

export type Locale = 'vi' | 'en' | 'fr' | 'zh-CN' | 'th' | 'lo' | 'ru'

const locales: Record<Locale, Record<string, string>> = { vi, en, fr, 'zh-CN': zhCN, th, lo, ru }

export function translate(locale: Locale, key: string, fallback = key): string {
  return locales[locale]?.[key] ?? locales.en?.[key] ?? fallback
}
