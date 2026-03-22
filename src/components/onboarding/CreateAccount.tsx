import { useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

export function CreateAccount({ onDone }: { onDone: () => void }) {
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
    setLoading(true)
    setError('')
    try {
      await callEngine('auth.register', { username, password })
      const result = await callEngine<{ token: string }>('auth.login', { username, password })
      const me = await callEngine<{ user_id: number; username: string }>('auth.me', { token: result.token })
      setAuth(result.token, me.username, me.user_id)
      onDone()
    } catch (err: unknown) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col items-center justify-center h-screen bg-surface text-white">
      <h1 className="text-2xl font-bold mb-6">Chào mừng đến Open Prompt</h1>
      <form onSubmit={handleSubmit} className="flex flex-col gap-3 w-80">
        <input
          autoFocus
          className="bg-white/10 rounded-lg px-4 py-2 outline-none focus:ring-2 ring-accent"
          placeholder="Tên đăng nhập"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          required
        />
        <input
          type="password"
          className="bg-white/10 rounded-lg px-4 py-2 outline-none focus:ring-2 ring-accent"
          placeholder="Mật khẩu (ít nhất 8 ký tự)"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
        {error && <p className="text-red-400 text-sm">{error}</p>}
        <button
          type="submit"
          disabled={loading}
          className="bg-accent rounded-lg py-2 font-semibold disabled:opacity-50 hover:bg-indigo-500 transition"
        >
          {loading ? 'Đang tạo...' : 'Tạo tài khoản'}
        </button>
      </form>
    </div>
  )
}
