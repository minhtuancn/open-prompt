import { useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

interface Props {
  onNext: () => void
}

/** AccountStep — tạo tài khoản (tách từ CreateAccount cũ) */
export function AccountStep({ onNext }: Props) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const setAuth = useAuthStore((s) => s.setAuth)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (password.length < 8) {
      setError('Mật khẩu cần ít nhất 8 ký tự')
      return
    }
    if (!/[A-Z]/.test(password) || !/[a-z]/.test(password) || !/[0-9]/.test(password)) {
      setError('Mật khẩu phải chứa chữ hoa, chữ thường và số')
      return
    }
    setLoading(true)
    setError('')
    try {
      await callEngine('auth.register', { username, password })
      const result = await callEngine<{ token: string }>('auth.login', { username, password })
      const me = await callEngine<{ user_id: number; username: string }>('auth.me', { token: result.token })
      setAuth(result.token, me.username, me.user_id)
      onNext()
    } catch (err: unknown) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <h2 className="text-lg font-semibold text-white text-center">Tạo tài khoản</h2>
      <p className="text-xs text-white/40 text-center">Dùng để lưu settings và lịch sử queries</p>
      <form onSubmit={handleSubmit} className="flex flex-col gap-3">
        <input
          autoFocus
          className="bg-white/10 rounded-lg px-4 py-2 text-sm text-white outline-none focus:ring-2 ring-indigo-500/50"
          placeholder="Tên đăng nhập"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          required
        />
        <input
          type="password"
          className="bg-white/10 rounded-lg px-4 py-2 text-sm text-white outline-none focus:ring-2 ring-indigo-500/50"
          placeholder="Mật khẩu (ít nhất 8 ký tự)"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
        {error && <p className="text-red-400 text-xs">{error}</p>}
        <button
          type="submit"
          disabled={loading}
          className="bg-indigo-500 hover:bg-indigo-400 rounded-lg py-2 text-sm font-medium text-white disabled:opacity-50 transition-colors"
        >
          {loading ? 'Đang tạo...' : 'Tiếp tục'}
        </button>
      </form>
    </div>
  )
}
