import { create } from 'zustand'

interface OverlayState {
  input: string
  chunks: string[]
  isStreaming: boolean
  error: string | null
  activeProvider: string | null
  activeModel: string | null
  fallbackProviders: string[]
  lastQuery: string
  setInput: (input: string) => void
  appendChunk: (chunk: string) => void
  setStreaming: (v: boolean) => void
  setError: (e: string | null) => void
  setActiveProvider: (p: string | null) => void
  setActiveModel: (m: string | null) => void
  setFallbackProviders: (providers: string[]) => void
  setLastQuery: (q: string) => void
  reset: () => void
}

export const useOverlayStore = create<OverlayState>()((set) => ({
  input: '',
  chunks: [],
  isStreaming: false,
  error: null,
  activeProvider: null,
  activeModel: null,
  fallbackProviders: [],
  lastQuery: '',
  setInput: (input) => set({ input }),
  appendChunk: (chunk) => set((s) => ({ chunks: [...s.chunks, chunk] })),
  setStreaming: (isStreaming) => set({ isStreaming }),
  setError: (error) => set({ error }),
  setActiveProvider: (activeProvider) => set({ activeProvider }),
  setActiveModel: (activeModel) => set({ activeModel }),
  setFallbackProviders: (fallbackProviders) => set({ fallbackProviders }),
  setLastQuery: (lastQuery) => set({ lastQuery }),
  reset: () => set({ input: '', chunks: [], isStreaming: false, error: null, fallbackProviders: [], lastQuery: '' }),
}))
