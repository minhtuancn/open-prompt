import { useState } from 'react'
import { ProvidersTab } from './ProvidersTab'
import { HotkeyTab } from './HotkeyTab'
import { AppearanceTab } from './AppearanceTab'
import { LanguageTab } from './LanguageTab'
import { SkillList } from '../skills/SkillList'
import { SkillEditor } from '../skills/SkillEditor'
import { UsageStats } from '../analytics/UsageStats'

type Tab = 'providers' | 'skills' | 'hotkey' | 'appearance' | 'language' | 'analytics'

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
  { id: 'skills', label: 'Skills' },
  { id: 'hotkey', label: 'Phím tắt' },
  { id: 'appearance', label: 'Giao diện' },
  { id: 'language', label: 'Ngôn ngữ' },
  { id: 'analytics', label: 'Thống kê' },
]

export function SettingsLayout({ onClose }: Props) {
  const [activeTab, setActiveTab] = useState<Tab>('providers')
  const [editingSkill, setEditingSkill] = useState<SkillData | undefined>()
  const [isNewSkill, setIsNewSkill] = useState(false)
  const [skillRefresh, setSkillRefresh] = useState(0)

  const handleSkillSave = () => {
    setEditingSkill(undefined)
    setIsNewSkill(false)
    setSkillRefresh((n) => n + 1)
  }

  const handleTabChange = (tab: Tab) => {
    setActiveTab(tab)
    setEditingSkill(undefined)
    setIsNewSkill(false)
  }

  return (
    <div className="flex flex-col" style={{ maxHeight: '600px' }}>
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
        {activeTab === 'skills' && (
          (editingSkill || isNewSkill) ? (
            <SkillEditor
              skill={editingSkill}
              onSave={handleSkillSave}
              onCancel={() => { setEditingSkill(undefined); setIsNewSkill(false) }}
            />
          ) : (
            <SkillList
              onEdit={(skill) => setEditingSkill(skill as SkillData)}
              onNew={() => setIsNewSkill(true)}
              refreshSignal={skillRefresh}
            />
          )
        )}
        {activeTab === 'hotkey' && <HotkeyTab />}
        {activeTab === 'appearance' && <AppearanceTab />}
        {activeTab === 'language' && <LanguageTab />}
        {activeTab === 'analytics' && <UsageStats />}
      </div>
    </div>
  )
}
