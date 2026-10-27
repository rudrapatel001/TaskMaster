import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import type { Task } from '../lib/tasks'
import { TaskCard } from './TaskCard'

/**
 * A card that can be dragged. Click (or press Enter) to open it; press Space to
 * pick it up with the keyboard, the arrow keys to move it, and Space to drop it.
 */
export function SortableTaskCard({
  task,
  assigneeName,
  done,
  today,
  draggable,
  onOpen,
}: {
  task: Task
  assigneeName?: string
  done: boolean
  today: string
  draggable: boolean
  onOpen: () => void
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: task.id,
    disabled: !draggable,
  })

  return (
    <li>
      <div
        ref={setNodeRef}
        style={{ transform: CSS.Transform.toString(transform), transition, opacity: isDragging ? 0.4 : 1 }}
        {...attributes}
        {...listeners}
        onClick={onOpen}
        onKeyDown={(event) => {
          if (event.key === 'Enter') onOpen()
          else listeners?.onKeyDown?.(event) // Space and Escape belong to dragging
        }}
        aria-label={`${task.identifier}: ${task.title}`}
        className={`rounded-lg focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-400 ${
          draggable ? 'cursor-grab active:cursor-grabbing' : 'cursor-pointer'
        }`}
      >
        <TaskCard task={task} assigneeName={assigneeName} done={done} today={today} />
      </div>
    </li>
  )
}