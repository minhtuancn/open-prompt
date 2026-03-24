import { create } from 'zustand'
import { callEngine } from '../hooks/useEngine'

export interface ProviderInfo {
  id: string
  name: string
  connected: boolean
}

interface ProviderState {
  providers: ProviderInfo[]
  loading: boolean
  error: string | null
  lastFetchToken: string | null
  fetch: (token: string) => Promise<void>
}

export const useProviderStore = create<ProviderState>()((set, get) => ({
  providers: [],
  loading: false,
  error: null,
  lastFetchToken: null,
  fetch: async (token: string) => {
    // Chỉ fetch lại nếu token thay đổi hoặc chưa fetch
    if (get().lastFetchToken === token && get().providers.length > 0) return
    set({ loading: true, error: null })
    try {
      const list = await callEngine<ProviderInfo[]>('providers.list', { token })
      set({ providers: list ?? [], loading: false, lastFetchToken: token })
    } catch (e) {
      console.error(e)
      set({ error: 'Không thể tải danh sách provider', loading: false })
    }
  },
}))
