import type { Task } from '../lib/tasks'

const styles: Record<Task['priority'], string> = {
  urgent: 'bg-red-500/15 text-red-300',
  high: 'bg-orange-500/15 text-orange-300',
  medium: 'bg-sky-500/15 text-sky-300',
  low: 'bg-slate-500/20 text-slate-300',
}

/** A small label for a task's priority. */
export function PriorityBadge({ priority }: { priority: Task['priority'] }) {
  return <span className={`rounded px-1.5 py-0.5 text-xs capitalize ${styles[priority]}`}>{priority}</span>
}