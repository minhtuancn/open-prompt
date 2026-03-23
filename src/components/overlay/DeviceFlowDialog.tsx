import { useEffect, useRef, useState } from 'react'
import { invoke } from '@tauri-apps/api/core'

interface Props {
  provider: string
  userCode: string
  verificationUri: string
  deviceCode: string
  onComplete: () => void
  onCancel: () => void
}

export function DeviceFlowDialog({ provider, userCode, verificationUri, deviceCode, onComplete, onCancel }: Props) {
  const [status, setStatus] = useState<'waiting' | 'success' | 'error'>('waiting')
  const [error, setError] = useState('')
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  useEffect(() => {
    // Poll mỗi 5 giây
    intervalRef.current = setInterval(async () => {
      try {
        const result = await invoke<{ done: boolean; error?: string }>('poll_oauth', {
          provider,
          deviceCode,
        })
        if (result.done) {
          if (result.error) {
            setStatus('error')
            setError(result.error)
          } else {
            setStatus('success')
            onComplete()
          }
          if (intervalRef.current) clearInterval(intervalRef.current)
        }
      } catch (e) {
        console.error('poll error:', e)
      }
    }, 5000)

    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [provider, deviceCode, onComplete])

  const handleOpenUrl = () => {
    window.open(verificationUri, '_blank')
  }

  return (
    <div className="bg-surface border border-white/10 rounded-xl p-4 shadow-xl">
      <div className="text-sm font-medium text-white mb-3">Đăng nhập {provider}</div>

      {status === 'waiting' && (
        <>
          <p className="text-xs text-white/60 mb-3">
            Mở <button onClick={handleOpenUrl} className="text-indigo-400 underline">{verificationUri}</button> và nhập code:
          </p>
          <div className="bg-black/30 border border-white/20 rounded-lg p-3 text-center mb-3">
            <span className="text-2xl font-mono font-bold text-white tracking-widest">{userCode}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="animate-pulse text-xs text-white/40">Đang chờ xác nhận...</span>
            <button onClick={onCancel} className="ml-auto text-xs text-white/30 hover:text-white/60">Hủy</button>
          </div>
        </>
      )}

      {status === 'success' && (
        <p className="text-green-400 text-sm">✓ Đăng nhập thành công!</p>
      )}

      {status === 'error' && (
        <p className="text-red-400 text-sm">✗ {error}</p>
      )}
    </div>
  )
}
