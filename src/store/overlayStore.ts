import { create } from 'zustand'

interface OverlayState {
  input: string
  chunks: string[]
  isStreaming: boolean
  error: string | null
  setInput: (input: string) => void
  appendChunk: (chunk: string) => void
  setStreaming: (v: boolean) => void
  setError: (e: string | null) => void
  reset: () => void
}

export const useOverlayStore = create<OverlayState>()((set) => ({
  input: '',
  chunks: [],
  isStreaming: false,
  error: null,
  setInput: (input) => set({ input }),
  appendChunk: (chunk) => set((s) => ({ chunks: [...s.chunks, chunk] })),
  setStreaming: (isStreaming) => set({ isStreaming }),
  setError: (error) => set({ error }),
  reset: () => set({ input: '', chunks: [], isStreaming: false, error: null }),
}))
