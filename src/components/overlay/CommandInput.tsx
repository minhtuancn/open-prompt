import { useEffect, useRef, useState } from 'react'
import { useOverlayStore } from '../../store/overlayStore'
import { SlashMenu, type SlashCommand } from './SlashMenu'

interface Props {
  onSubmit: (input: string, slashName?: string, extraVars?: Record<string, string>) => void
}

export function CommandInput({ onSubmit }: Props) {
  const { input, setInput, isStreaming } = useOverlayStore()

  const [commands, setCommands] = useState<SlashCommand[]>([])
  const [slashMenuVisible, setSlashMenuVisible] = useState(false)
  const [slashQuery, setSlashQuery] = useState('')
  const [selectedCmd, setSelectedCmd] = useState<SlashCommand | null>(null)
  const [extraVars, setExtraVars] = useState<Record<string, string>>({})
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    const token = localStorage.getItem('auth_token')
    if (!token) return
    window.__rpc?.call('commands.list', { token })
      .then((res: unknown) => {
        const data = res as { commands: SlashCommand[] }
        setCommands(data.commands || [])
      })
      .catch(console.error)
  }, [])

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const value = e.target.value
    setInput(value)
    if (selectedCmd) return
    if (value.startsWith('/') && !value.includes('\n')) {
      setSlashQuery(value.slice(1))
      setSlashMenuVisible(true)
    } else {
      setSlashMenuVisible(false)
      setSlashQuery('')
    }
  }

  const handleSlashSelect = (cmd: SlashCommand) => {
    setSelectedCmd(cmd)
    setSlashMenuVisible(false)
    setInput('')
    const initVars: Record<string, string> = {}
    cmd.required_vars.forEach(v => { initVars[v] = '' })
    setExtraVars(initVars)
    textareaRef.current?.focus()
  }

  const handleCloseMenu = () => setSlashMenuVisible(false)

  const handleClearSlash = () => {
    setSelectedCmd(null)
    setExtraVars({})
    setInput('')
    setSlashMenuVisible(false)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (slashMenuVisible) return
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      if (!isStreaming) handleSubmit()
    }
    if (e.key === 'Escape') {
      if (selectedCmd) handleClearSlash()
      else window.close()
    }
  }

  const handleSubmit = () => {
    if (selectedCmd) {
      onSubmit(input.trim(), selectedCmd.slash_name, extraVars)
      setSelectedCmd(null)
      setExtraVars({})
    } else {
      if (input.trim()) onSubmit(input.trim())
    }
    setInput('')
  }

  return (
    <div className="relative">
      <SlashMenu
        commands={commands}
        query={slashQuery}
        onSelect={handleSlashSelect}
        onClose={handleCloseMenu}
        visible={slashMenuVisible}
      />

      {selectedCmd && (
        <div className="flex items-center gap-2 px-5 pt-3 pb-1">
          <span className="font-mono text-sm text-indigo-400 bg-indigo-500/20 px-2 py-0.5 rounded-md">
            /{selectedCmd.slash_name}
          </span>
          <span className="text-xs text-white/50">{selectedCmd.title}</span>
          <button
            onClick={handleClearSlash}
            className="ml-auto text-white/30 hover:text-white/60 text-xs"
          >
            ✕
          </button>
        </div>
      )}

      {selectedCmd && Object.keys(extraVars).length > 0 && (
        <div className="px-5 py-2 flex flex-col gap-1.5">
          {Object.keys(extraVars).map(varName => (
            <input
              key={varName}
              type="text"
              placeholder={varName}
              value={extraVars[varName]}
              onChange={e => setExtraVars(prev => ({ ...prev, [varName]: e.target.value }))}
              className="bg-white/5 border border-white/10 rounded-lg px-3 py-1.5 text-sm text-white placeholder-white/30 outline-none focus:border-indigo-500/50"
            />
          ))}
        </div>
      )}

      <textarea
        ref={textareaRef}
        value={input}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        placeholder={selectedCmd ? `Nhập nội dung cho /${selectedCmd.slash_name}...` : 'Hỏi AI bất cứ điều gì... (/ để dùng slash command)'}
        rows={2}
        disabled={isStreaming}
        className="w-full bg-transparent text-white placeholder-white/30 resize-none outline-none px-5 py-4 text-sm leading-relaxed disabled:opacity-50"
      />

      <div className="px-5 pb-4 flex items-center justify-between">
        <span className="text-xs text-white/20">
          {isStreaming ? 'Đang xử lý...' : 'Enter để gửi • Shift+Enter xuống dòng • / để slash command'}
        </span>
        <button
          onClick={handleSubmit}
          disabled={isStreaming || (!input.trim() && !selectedCmd)}
          className="px-4 py-1.5 bg-indigo-500/80 hover:bg-indigo-500 text-white text-xs font-medium rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {isStreaming ? '⟳' : 'Gửi'}
        </button>
      </div>
    </div>
  )
}
