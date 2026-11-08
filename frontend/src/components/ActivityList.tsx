import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import type { ActivityEvent } from '../lib/activity'

function text(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0].toUpperCase())
    .join('')
}

function when(iso: string): string {
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}

// How the field names the backend uses read in a sentence.
const fieldNames: Record<string, string> = {
  title: 'title',
  description: 'description',
  priority: 'priority',
  assignee_id: 'assignee',
  due_date: 'due date',
}

/** "priority, due date" from the list of fields an edit changed. Empty if unknown. */
function changedFields(value: unknown): string {
  if (!Array.isArray(value)) return ''
  return value
    .filter((field): field is string => typeof field === 'string')
    .map((field) => fieldNames[field] ?? field)
    .join(', ')
}

/** "WEB-12 Fix login", as a link that opens that task on its board when it can. */
function TaskRef({ event, workspaceSlug }: { event: ActivityEvent; workspaceSlug?: string }) {
  const identifier = text(event.metadata.identifier)
  const title = text(event.metadata.title)
  const label = (
    <>
      <strong className="font-medium">{identifier}</strong>
      {title && <> {title}</>}
    </>
  )

  // "WEB-12" -> project key "WEB", task number 12. A deleted task has nowhere to link to.
  const match = /^([A-Z][A-Z0-9]*)-(\d+)$/.exec(identifier)
  if (!workspaceSlug || !match || event.action === 'task.deleted') return label
  return (
    <Link to={`/w/${workspaceSlug}/projects/${match[1]}?task=${match[2]}`} className="hover:underline">
      {label}
    </Link>
  )
}

/** The sentence after the person's name. */
function describe(event: ActivityEvent, subject: ReactNode): ReactNode {
  switch (event.action) {
    case 'task.created':
      return <>created {subject}</>
    case 'task.moved':
      return (
        <>
          moved {subject} to <strong className="font-medium">{text(event.metadata.to)}</strong>
        </>
      )
    case 'task.updated': {
      const fields = changedFields(event.metadata.fields)
      return (
        <>
          updated {subject}
          {fields && <span className="text-slate-400"> ({fields})</span>}
        </>
      )
    }
    case 'task.deleted':
      return <>deleted {subject}</>
    case 'task.commented': {
      const excerpt = text(event.metadata.excerpt)
      return (
        <>
          commented on {subject}
          {excerpt && <span className="text-slate-400">: “{excerpt}”</span>}
        </>
      )
    }
    default:
      return <>{event.action}</> // an event type this version does not know yet
  }
}

/**
 * A list of feed entries. In the "workspace" scope each entry names its task;
 * in the "task" scope it says "this task", since you are already looking at it.
 */
export function ActivityList({
  events,
  scope,
  workspaceSlug,
  hasMore,
  loadingMore,
  onLoadMore,
}: {
  events: ActivityEvent[]
  scope: 'workspace' | 'task'
  workspaceSlug?: string
  hasMore: boolean
  loadingMore: boolean
  onLoadMore: () => void
}) {
  return (
    <div>
      <ul className="space-y-4">
        {events.map((event) => (
          <li key={event.id} className="flex gap-3">
            <span
              aria-hidden="true"
              className="mt-0.5 grid h-8 w-8 shrink-0 place-items-center rounded-full bg-indigo-500/30 text-xs font-medium"
            >
              {initials(event.actor_name)}
            </span>
            <div className="min-w-0">
              <p className="text-sm">
                <span className="font-medium">{event.actor_name}</span>{' '}
                {describe(event, scope === 'task' ? 'this task' : <TaskRef event={event} workspaceSlug={workspaceSlug} />)}
              </p>
              <p className="text-xs text-slate-500">{when(event.created_at)}</p>
            </div>
          </li>
        ))}
      </ul>

      {hasMore && (
        <button
          onClick={onLoadMore}
          disabled={loadingMore}
          className="mt-5 rounded-md bg-slate-800 px-4 py-2 text-sm hover:bg-slate-700 disabled:opacity-60"
        >
          {loadingMore ? 'Loading…' : 'Load older activity'}
        </button>
      )}
    </div>
  )
}