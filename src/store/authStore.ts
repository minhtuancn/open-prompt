import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface AuthState {
  token: string | null
  username: string | null
  userId: number | null
  setAuth: (token: string, username: string, userId: number) => void
  clearAuth: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      username: null,
      userId: null,
      setAuth: (token, username, userId) => set({ token, username, userId }),
      clearAuth: () => set({ token: null, username: null, userId: null }),
    }),
    { name: 'op-auth' }
  )
)
