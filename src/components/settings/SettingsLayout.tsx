import { useState } from 'react'
import { ProvidersTab } from './ProvidersTab'
import { HotkeyTab } from './HotkeyTab'
import { AppearanceTab } from './AppearanceTab'
import { LanguageTab } from './LanguageTab'
import { SkillList } from '../skills/SkillList'
import { SkillEditor } from '../skills/SkillEditor'
import { UsageStats } from '../analytics/UsageStats'
import { PromptList } from '../prompts/PromptList'
import { PromptEditor } from '../prompts/PromptEditor'
import { HistoryPanel } from '../history/HistoryPanel'
import { UpdateTab } from './UpdateTab'
import { MarketplaceTab } from './MarketplaceTab'

type Tab = 'providers' | 'prompts' | 'skills' | 'history' | 'marketplace' | 'hotkey' | 'appearance' | 'language' | 'analytics' | 'update'

interface SkillData {
  id?: number
  name: string
  prompt_text: string
  model: string
  provider: string
  tags: string
}

interface Props {
  onClose: () => void
}

const TABS: { id: Tab; label: string }[] = [
  { id: 'providers', label: 'Providers' },
  { id: 'prompts', label: 'Prompts' },
  { id: 'skills', label: 'Skills' },
  { id: 'history', label: 'Lịch sử' },
  { id: 'marketplace', label: 'Marketplace' },
  { id: 'hotkey', label: 'Phím tắt' },
  { id: 'appearance', label: 'Giao diện' },
  { id: 'language', label: 'Ngôn ngữ' },
  { id: 'analytics', label: 'Thống kê' },
  { id: 'update', label: 'Cập nhật' },
]

export function SettingsLayout({ onClose }: Props) {
  const [activeTab, setActiveTab] = useState<Tab>('providers')
  const [skillView, setSkillView] = useState<SkillData | 'new' | null>(null)
  const [skillRefresh, setSkillRefresh] = useState(0)
  const [promptView, setPromptView] = useState<{ id?: number; title: string; content: string; category: string; tags: string; is_slash: boolean; slash_name: string } | 'new' | null>(null)
  const [promptRefresh, setPromptRefresh] = useState(0)

  const handleSkillSave = () => {
    setSkillView(null)
    setSkillRefresh((n) => n + 1)
  }

  const handlePromptSave = () => {
    setPromptView(null)
    setPromptRefresh((n) => n + 1)
  }

  const handleTabChange = (tab: Tab) => {
    setActiveTab(tab)
    setSkillView(null)
    setPromptView(null)
  }

  return (
    <div className="flex flex-col max-h-[600px]">
      <div className="flex items-center justify-between px-5 py-3 border-b border-white/10 shrink-0">
        <span className="text-sm font-semibold text-white">Cài đặt</span>
        <button onClick={onClose} className="text-white/40 hover:text-white transition-colors text-xl leading-none">×</button>
      </div>

      <div className="flex gap-1 px-4 py-2 border-b border-white/10 overflow-x-auto shrink-0">
        {TABS.map((tab) => (
          <button
            key={tab.id}
            onClick={() => handleTabChange(tab.id)}
            className={`text-xs px-3 py-1.5 rounded-lg whitespace-nowrap transition-colors ${activeTab === tab.id ? 'bg-indigo-500/20 text-indigo-300' : 'text-white/40 hover:text-white/70'}`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="flex-1 overflow-y-auto px-5 py-4">
        {activeTab === 'providers' && <ProvidersTab />}
        {activeTab === 'prompts' && (
          promptView !== null ? (
            <PromptEditor
              initial={promptView === 'new' ? undefined : promptView}
              onSave={(_p: unknown) => handlePromptSave()}
              onCancel={() => setPromptView(null)}
            />
          ) : (
            <PromptList
              onEdit={(prompt) => setPromptView(prompt)}
              key={promptRefresh}
            />
          )
        )}
        {activeTab === 'history' && <HistoryPanel />}
        {activeTab === 'marketplace' && <MarketplaceTab />}
        {activeTab === 'skills' && (
          skillView !== null ? (
            <SkillEditor
              skill={skillView === 'new' ? undefined : skillView}
              onSave={handleSkillSave}
              onCancel={() => setSkillView(null)}
            />
          ) : (
            <SkillList
              onEdit={(skill) => setSkillView(skill as SkillData)}
              onNew={() => setSkillView('new')}
              refreshSignal={skillRefresh}
            />
          )
        )}
        {activeTab === 'hotkey' && <HotkeyTab />}
        {activeTab === 'appearance' && <AppearanceTab />}
        {activeTab === 'language' && <LanguageTab />}
        {activeTab === 'analytics' && <UsageStats />}
        {activeTab === 'update' && <UpdateTab />}
      </div>
    </div>
  )
}
