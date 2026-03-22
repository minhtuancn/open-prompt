import { invoke } from '@tauri-apps/api/core'
import { listen } from '@tauri-apps/api/event'

/** callEngine gọi Go Engine qua Tauri IPC */
export async function callEngine<T>(method: string, params: Record<string, unknown>): Promise<T> {
  return invoke<T>('call_engine', { method, params })
}

/** streamQuery gọi query.stream và subscribe notifications */
export async function streamQuery(
  params: { token: string; input: string; model?: string },
  onChunk: (chunk: string) => void,
  onDone: () => void,
  onError: (err: string) => void,
): Promise<void> {
  const unlisten = await listen<{ delta: string; done: boolean; error?: string }>(
    'stream-chunk',
    (event) => {
      const { delta, done, error } = event.payload
      if (error) {
        onError(error)
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

  // Trigger stream (fire and forget — response arrives via events)
  callEngine('query.stream', params).catch(onError)
}
