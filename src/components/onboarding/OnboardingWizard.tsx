import { useState } from 'react'
import { AccountStep } from './AccountStep'
import { WelcomeStep } from './WelcomeStep'
import { ProviderStep } from './ProviderStep'
import { HotkeyStep } from './HotkeyStep'
import { DoneStep } from './DoneStep'

interface Props {
  onComplete: () => void
}

const TOTAL_STEPS = 5

/** OnboardingWizard — wizard 5 bước cho first-run experience */
export function OnboardingWizard({ onComplete }: Props) {
  const [step, setStep] = useState(0)

  const renderStep = () => {
    switch (step) {
      case 0: return <WelcomeStep onNext={() => setStep(1)} />
      case 1: return <AccountStep onNext={() => setStep(2)} />
      case 2: return <ProviderStep onNext={() => setStep(3)} onSkip={() => setStep(3)} />
      case 3: return <HotkeyStep onNext={() => setStep(4)} />
      case 4: return <DoneStep onComplete={onComplete} />
      default: return null
    }
  }

  return (
    <div className="flex items-center justify-center h-screen bg-surface">
      <div className="bg-[#1a1a2e] border border-white/10 rounded-2xl p-8 w-[480px] max-h-[90vh] overflow-y-auto shadow-2xl">
        {/* Progress dots */}
        <div className="flex justify-center gap-2 mb-6">
          {Array.from({ length: TOTAL_STEPS }, (_, i) => (
            <div
              key={i}
              className={`w-2 h-2 rounded-full transition-colors ${
                i === step ? 'bg-indigo-500' : i < step ? 'bg-indigo-500/40' : 'bg-white/20'
              }`}
            />
          ))}
        </div>
        {renderStep()}
      </div>
    </div>
  )
}
