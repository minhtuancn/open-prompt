import vi from './locales/vi.json'
import en from './locales/en.json'

export type Locale = 'vi' | 'en'

const locales: Record<Locale, Record<string, string>> = { vi, en }

export function translate(locale: Locale, key: string, fallback = key): string {
  return locales[locale]?.[key] ?? locales.vi?.[key] ?? fallback
}
