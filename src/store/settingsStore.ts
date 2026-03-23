import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { Locale } from '../i18n'

type FontSize = 'sm' | 'base' | 'lg'

interface SettingsState {
  locale: Locale
  fontSize: FontSize
  setLocale: (locale: Locale) => void
  setFontSize: (size: FontSize) => void
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      locale: 'vi',
      fontSize: 'base',
      setLocale: (locale) => set({ locale }),
      setFontSize: (fontSize) => set({ fontSize }),
    }),
    { name: 'open-prompt-settings' }
  )
)
