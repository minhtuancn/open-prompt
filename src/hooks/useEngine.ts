import { invoke } from '@tauri-apps/api/core'
import { listen } from '@tauri-apps/api/event'

/** callEngine gọi Go Engine qua Tauri IPC */
export async function callEngine<T>(method: string, params: Record<string, unknown>): Promise<T> {
  return invoke<T>('call_engine', { method, params })
}

interface StreamChunkPayload {
  delta: string
  done: boolean
  error?: string
  error_message?: string
  fallback_providers?: string[]
}

/** streamQuery gọi query.stream và subscribe notifications */
export async function streamQuery(
  params: { token: string; input: string; model?: string; provider?: string },
  onChunk: (chunk: string) => void,
  onDone: () => void,
  onError: (err: string, fallbackProviders?: string[]) => void,
): Promise<void> {
  const unlisten = await listen<StreamChunkPayload>(
    'stream-chunk',
    (event) => {
      const { delta, done, error, error_message, fallback_providers } = event.payload
      if (error) {
        onError(error_message || error, fallback_providers)
        unlisten()
        return
      }
      if (done) {
        onDone()
        unlisten()
        return
      }
      onChunk(delta)
    }
  )

  callEngine('query.stream', params).catch((e) => onError(String(e)))
}
