import { useEffect, useState } from 'react'
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

interface PriorityItem {
  id: number
  priority: number
  provider: string
  model: string
  is_enabled: boolean
}

/** SortableItem — mỗi item trong danh sách kéo thả */
function SortableItem({ item, onToggle }: { item: PriorityItem; onToggle: (id: number) => void }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: item.id,
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`flex items-center gap-3 px-3 py-2 rounded-lg border transition-colors ${
        item.is_enabled
          ? 'bg-white/5 border-white/10'
          : 'bg-white/2 border-white/5 opacity-50'
      }`}
    >
      {/* Drag handle */}
      <button
        {...attributes}
        {...listeners}
        className="cursor-grab active:cursor-grabbing text-white/30 hover:text-white/60 shrink-0"
        title="Kéo để sắp xếp"
      >
        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
          <circle cx="5" cy="3" r="1.5" />
          <circle cx="11" cy="3" r="1.5" />
          <circle cx="5" cy="8" r="1.5" />
          <circle cx="11" cy="8" r="1.5" />
          <circle cx="5" cy="13" r="1.5" />
          <circle cx="11" cy="13" r="1.5" />
        </svg>
      </button>

      {/* Priority number */}
      <span className="text-xs text-white/30 font-mono w-5 text-center shrink-0">
        {item.priority}
      </span>

      {/* Provider + Model */}
      <div className="flex-1 min-w-0">
        <span className="text-sm text-white">{item.provider}</span>
        <span className="text-xs text-white/40 ml-2">{item.model}</span>
      </div>

      {/* Toggle enabled */}
      <button
        onClick={() => onToggle(item.id)}
        className={`text-xs px-2 py-0.5 rounded-full transition-colors ${
          item.is_enabled
            ? 'bg-green-500/20 text-green-400 hover:bg-green-500/30'
            : 'bg-white/10 text-white/30 hover:bg-white/20'
        }`}
      >
        {item.is_enabled ? 'ON' : 'OFF'}
      </button>
    </div>
  )
}

/** ModelPriorityList — danh sách kéo thả thứ tự ưu tiên model */
export function ModelPriorityList() {
  const token = useAuthStore((s) => s.token)
  const [items, setItems] = useState<PriorityItem[]>([])
  const [saving, setSaving] = useState(false)

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  )

  // Load priority list từ providers.list
  useEffect(() => {
    if (!token) return
    callEngine<{ id: string; name: string; connected: boolean }[]>('providers.list', { token })
      .then((providers) => {
        if (!providers) return
        const connected = providers.filter((p) => p.connected)
        setItems(
          connected.map((p, i) => ({
            id: i + 1,
            priority: i + 1,
            provider: p.id,
            model: p.name,
            is_enabled: true,
          }))
        )
      })
      .catch(console.error)
  }, [token])

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event
    if (!over || active.id === over.id) return

    const oldIndex = items.findIndex((i) => i.id === active.id)
    const newIndex = items.findIndex((i) => i.id === over.id)
    const reordered = arrayMove(items, oldIndex, newIndex).map((item, i) => ({
      ...item,
      priority: i + 1,
    }))
    setItems(reordered)

    // Lưu thứ tự mới
    if (!token) return
    setSaving(true)
    try {
      await callEngine('providers.set_priority', {
        token,
        priorities: reordered.map((item) => ({
          provider: item.provider,
          model: item.model,
          priority: item.priority,
          is_enabled: item.is_enabled,
        })),
      })
    } catch (e) {
      console.error(e)
    } finally {
      setSaving(false)
    }
  }

  const handleToggle = async (id: number) => {
    const updated = items.map((item) =>
      item.id === id ? { ...item, is_enabled: !item.is_enabled } : item
    )
    setItems(updated)

    if (!token) return
    try {
      await callEngine('providers.set_priority', {
        token,
        priorities: updated.map((item) => ({
          provider: item.provider,
          model: item.model,
          priority: item.priority,
          is_enabled: item.is_enabled,
        })),
      })
    } catch (e) {
      console.error(e)
    }
  }

  if (items.length === 0) {
    return (
      <p className="text-xs text-white/30">
        Chưa có provider nào được kết nối. Thêm API key ở trên.
      </p>
    )
  }

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center justify-between mb-1">
        <span className="text-xs text-white/40">Kéo thả để sắp xếp thứ tự ưu tiên</span>
        {saving && <span className="text-xs text-indigo-400 animate-pulse">Đang lưu...</span>}
      </div>

      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
        <SortableContext items={items.map((i) => i.id)} strategy={verticalListSortingStrategy}>
          {items.map((item) => (
            <SortableItem key={item.id} item={item} onToggle={handleToggle} />
          ))}
        </SortableContext>
      </DndContext>
    </div>
  )
}
