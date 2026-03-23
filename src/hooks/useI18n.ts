import { useSettingsStore } from '../store/settingsStore'
import { translate } from '../i18n'

export function useI18n() {
  const locale = useSettingsStore((s) => s.locale)
  const t = (key: string, fallback?: string) => translate(locale, key, fallback)
  return { t, locale }
}
