import type { Task } from '../lib/tasks'
import { PriorityBadge } from './PriorityBadge'

function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0].toUpperCase())
    .join('')
}

/** The look of one card. (The dragging behaviour is added by SortableTaskCard.) */
export function TaskCard({
  task,
  assigneeName,
  done,
  today,
}: {
  task: Task
  assigneeName?: string
  done: boolean
  today: string // "2026-10-20": dates in this format compare correctly as text
}) {
  const overdue = !done && task.due_date !== null && task.due_date < today

  return (
    <div className="rounded-lg bg-slate-800 p-3 shadow ring-1 ring-slate-700">
      <p className="text-xs text-slate-400">{task.identifier}</p>
      <p className="mt-1 text-sm">{task.title}</p>

      <div className="mt-3 flex items-center gap-2 text-xs">
        <PriorityBadge priority={task.priority} />
        {task.due_date && (
          <span className={overdue ? 'text-red-300' : 'text-slate-400'}>
            {overdue ? 'Overdue · ' : 'Due '}
            {new Date(`${task.due_date}T00:00:00`).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}
          </span>
        )}
        {assigneeName && (
          <span
            title={assigneeName}
            aria-label={`Assigned to ${assigneeName}`}
            className="ml-auto grid h-6 w-6 place-items-center rounded-full bg-indigo-500/30 text-[10px] font-medium"
          >
            {initials(assigneeName)}
          </span>
        )}
      </div>
    </div>
  )
}