import { useEffect, useRef, useState } from 'react'
import { useOverlayStore } from '../../store/overlayStore'
import { useAuthStore } from '../../store/authStore'
import { SlashMenu, type SlashCommand } from './SlashMenu'
import { ModelPicker } from './ModelPicker'
import { MentionHint } from './MentionHint'

interface Props {
  onSubmit: (input: string, slashName?: string, extraVars?: Record<string, string>) => void
}

export function CommandInput({ onSubmit }: Props) {
  const { input, setInput, isStreaming, activeProvider, setActiveProvider } = useOverlayStore()
  const token = useAuthStore((s) => s.token)

  const [commands, setCommands] = useState<SlashCommand[]>([])
  const [slashMenuVisible, setSlashMenuVisible] = useState(false)
  const [slashQuery, setSlashQuery] = useState('')
  const [selectedCmd, setSelectedCmd] = useState<SlashCommand | null>(null)
  const [extraVars, setExtraVars] = useState<Record<string, string>>({})
  const [showModelPicker, setShowModelPicker] = useState(false)
  const [mentionQuery, setMentionQuery] = useState('')
  const [showMentionHint, setShowMentionHint] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    if (!token) return
    setError(null)
    window.__rpc?.call('commands.list', { token })
      .then((res: unknown) => {
        const data = res as { commands: SlashCommand[] }
        setCommands(data.commands || [])
      })
      .catch((e: unknown) => { console.error(e); setError('Không thể tải dữ liệu') })
  }, [])

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const value = e.target.value
    setInput(value)
    if (selectedCmd) return

    // Detect slash commands
    if (value.startsWith('/') && !value.includes('\n')) {
      setSlashQuery(value.slice(1))
      setSlashMenuVisible(true)
    } else {
      setSlashMenuVisible(false)
      setSlashQuery('')
    }

    // Detect @mention
    const atMatch = value.match(/@(\w*)$/)
    if (atMatch) {
      setMentionQuery(atMatch[1])
      setShowMentionHint(true)
    } else {
      setShowMentionHint(false)
      setMentionQuery('')
    }
  }

  const handleMentionSelect = (alias: string) => {
    const newInput = input.replace(/@\w*$/, '')
    setInput(newInput)
    setActiveProvider(alias)
    setShowMentionHint(false)
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

    // Ctrl+M → model picker
    if (e.key === 'm' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault()
      setShowModelPicker((v) => !v)
      return
    }

    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      if (!isStreaming) handleSubmit()
    }
    if (e.key === 'Escape') {
      if (showModelPicker) setShowModelPicker(false)
      else if (selectedCmd) handleClearSlash()
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
      {showModelPicker && (
        <ModelPicker
          onSelect={(name) => setActiveProvider(name)}
          onClose={() => setShowModelPicker(false)}
        />
      )}

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

      {error && <p className="text-red-400 text-sm px-3 py-2">{error}</p>}

      <div className="relative">
        <MentionHint query={mentionQuery} onSelect={handleMentionSelect} visible={showMentionHint} />
        <textarea
          ref={textareaRef}
          value={input}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          placeholder={selectedCmd ? `Nhập nội dung cho /${selectedCmd.slash_name}...` : 'Hỏi AI... (/ slash command • @ chọn provider • Ctrl+M model picker)'}
          rows={2}
          disabled={isStreaming}
          className="w-full bg-transparent text-white placeholder-white/30 resize-none outline-none px-5 py-4 text-sm leading-relaxed disabled:opacity-50"
        />
      </div>

      <div className="px-5 pb-4 flex items-center justify-between">
        <span className="text-xs text-white/20">
          {activeProvider && (
            <span className="text-indigo-400 mr-2">
              @{activeProvider}
              <button onClick={() => setActiveProvider(null)} className="ml-1 text-white/30 hover:text-white/60" aria-label="Xoá provider">✕</button>
            </span>
          )}
          {isStreaming ? 'Đang xử lý...' : 'Enter gửi • Ctrl+M chọn model • @ mention provider'}
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
