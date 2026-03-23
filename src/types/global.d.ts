// Khai báo global type cho window.__rpc (JSON-RPC client được inject bởi Tauri)
declare global {
  interface Window {
    __rpc?: {
      call: (method: string, params: unknown) => Promise<unknown>
    }
  }
}

export {}
