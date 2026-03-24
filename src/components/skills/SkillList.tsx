import { useEffect, useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

interface Skill {
  id: number
  name: string
  prompt_text: string
  model: string
  provider: string
  tags: string
}

interface Props {
  onEdit: (skill: Skill) => void
  onNew: () => void
  refreshSignal: number
}

export function SkillList({ onEdit, onNew, refreshSignal }: Props) {
  const token = useAuthStore((s) => s.token)
  const [skills, setSkills] = useState<Skill[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) return
    setLoading(true)
    setError(null)
    callEngine<{ skills: Skill[] }>('skills.list', { token })
      .then((res) => setSkills(res.skills ?? []))
      .catch((e) => { console.error(e); setError('Không thể tải dữ liệu') })
      .finally(() => setLoading(false))
  }, [refreshSignal, token])

  const handleDelete = async (id: number) => {
    if (!token || !confirm('Xóa skill này?')) return
    try {
      await callEngine('skills.delete', { token, id })
      setSkills((prev) => prev.filter((s) => s.id !== id))
    } catch (e) {
      alert('Xóa thất bại: ' + String(e))
    }
  }

  if (loading) {
    return <div className="text-white/40 text-sm text-center py-8">Đang tải...</div>
  }

  if (error) {
    return <p className="text-red-400 text-sm px-3 py-2">{error}</p>
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs text-white/40">{skills.length} skill</span>
        <button
          onClick={onNew}
          className="text-xs px-3 py-1.5 bg-indigo-500/80 hover:bg-indigo-500 text-white rounded-lg transition-colors"
        >
          + Tạo mới
        </button>
      </div>

      {skills.length === 0 && (
        <div className="text-white/30 text-sm text-center py-6">Chưa có skill. Tạo skill đầu tiên!</div>
      )}

      {skills.map((skill) => (
        <div
          key={skill.id}
          className="bg-white/5 border border-white/10 rounded-xl p-3 flex items-start justify-between gap-2"
        >
          <div className="flex-1 min-w-0">
            <div className="font-medium text-sm text-white truncate">{skill.name}</div>
            {skill.prompt_text && (
              <div className="text-xs text-white/40 mt-0.5 line-clamp-2">{skill.prompt_text}</div>
            )}
            <div className="flex gap-1.5 mt-1 flex-wrap">
              {skill.provider && (
                <span className="text-xs text-indigo-400/70 bg-indigo-500/10 px-1.5 py-0.5 rounded">
                  {skill.provider}
                </span>
              )}
              {skill.tags && skill.tags.split(',').map((tag) => tag.trim()).filter(Boolean).map((tag) => (
                <span key={tag} className="text-xs text-white/30 bg-white/5 px-1.5 py-0.5 rounded">{tag}</span>
              ))}
            </div>
          </div>
          <div className="flex gap-1 shrink-0">
            <button onClick={() => onEdit(skill)} className="text-xs px-2 py-1 text-white/50 hover:text-white transition-colors">Sửa</button>
            <button onClick={() => handleDelete(skill.id)} className="text-xs px-2 py-1 text-red-400/60 hover:text-red-400 transition-colors">Xóa</button>
          </div>
        </div>
      ))}
    </div>
  )
}
