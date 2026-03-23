import { useState } from 'react'

export function UpdateTab() {
  const [checking, setChecking] = useState(false)
  const [status, setStatus] = useState('')

  const handleCheckUpdate = async () => {
    setChecking(true)
    setStatus('')
    try {
      // Tauri updater plugin — check via JS API
      const { check } = await import('@tauri-apps/plugin-updater')
      const update = await check()
      if (update) {
        setStatus(`Có bản cập nhật mới: ${update.version}`)
      } else {
        setStatus('Bạn đang dùng phiên bản mới nhất.')
      }
    } catch (e) {
      setStatus(`Không thể kiểm tra: ${e}`)
    } finally {
      setChecking(false)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="bg-white/5 border border-white/10 rounded-xl p-4">
        <div className="text-sm font-medium text-white mb-2">Phiên bản</div>
        <p className="text-xs text-white/50 mb-3">Open Prompt v0.3.0</p>
        <button
          onClick={handleCheckUpdate}
          disabled={checking}
          className="text-xs px-4 py-2 bg-indigo-500/80 hover:bg-indigo-500 text-white rounded-lg transition-colors disabled:opacity-40"
        >
          {checking ? 'Đang kiểm tra...' : 'Kiểm tra cập nhật'}
        </button>
        {status && <p className="text-xs text-white/60 mt-2">{status}</p>}
      </div>
    </div>
  )
}
