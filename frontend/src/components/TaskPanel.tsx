import { useEffect, useRef, useState } from 'react'
import { ApiError } from '../lib/api'
import type { Member } from '../lib/members'
import type { Status } from '../lib/projects'
import type { Task, TaskPatch } from '../lib/tasks'
import { useDeleteTask, useUpdateTask } from '../lib/tasks'
import { PriorityBadge } from './PriorityBadge'
import type { Role } from '../lib/permissions'
import { Comments } from './Comments'

const priorities: Task['priority'][] = ['low', 'medium', 'high', 'urgent']

const fieldClass =
  'mt-1 w-full rounded-md bg-slate-800 px-3 py-2 text-sm text-white outline-none ring-1 ring-slate-600 focus:ring-2 focus:ring-indigo-400 disabled:opacity-60'

/** A side panel for reading and editing one task. Fields save as soon as you leave them. */
export function TaskPanel({
  workspaceId,
  projectId,
  task,
  statuses,
  members,
  canWrite,
  role,
  myUserId,
  onMove,
  onClose,
}: {
  workspaceId: string
  projectId: string
  task: Task
  statuses: Status[]
  members: Member[]
  canWrite: boolean
  role: Role
  myUserId: string
  onMove: (statusId: string) => void
  onClose: () => void
}) {
  const update = useUpdateTask(workspaceId, projectId)
  const remove = useDeleteTask(workspaceId, projectId)
  const [error, setError] = useState<string | null>(null)
  const [confirmingDelete, setConfirmingDelete] = useState(false)
  const panelRef = useRef<HTMLElement>(null)

  // Focus the panel when it opens or switches task, and close it with Escape.
  useEffect(() => {
    panelRef.current?.focus()
  }, [task.id])
  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [onClose])

  async function save(patch: TaskPatch) {
    setError(null)
    try {
      await update.mutateAsync({ number: task.number, patch })
    } catch (err) {
      const firstFieldError = err instanceof ApiError ? Object.values(err.fields)[0] : undefined
      setError(firstFieldError ?? 'Could not save the change. Please try again.')
    }
  }

  async function onDelete() {
    setError(null)
    try {
      await remove.mutateAsync(task.number)
      onClose()
    } catch {
      setError('Could not delete the task. Please try again.')
    }
  }

  const assigneeIsKnown = task.assignee_id === null || members.some((member) => member.user_id === task.assignee_id)

  return (
    <aside
      ref={panelRef}
      tabIndex={-1}
      aria-label={`Task ${task.identifier}`}
      className="fixed inset-y-0 right-0 z-20 w-full max-w-md overflow-y-auto bg-slate-900 p-6 shadow-2xl ring-1 ring-slate-700 focus:outline-none"
    >
      <div className="flex items-center justify-between">
        <span className="text-sm text-slate-400">{task.identifier}</span>
        <button
          onClick={onClose}
          aria-label="Close"
          className="rounded-md px-2 py-1 text-slate-400 hover:bg-slate-800 hover:text-white"
        >
          ✕
        </button>
      </div>

      {error && (
        <p role="alert" className="mt-4 rounded-md bg-red-500/10 px-3 py-2 text-sm text-red-300">
          {error}
        </p>
      )}

      {/* "key" restarts these fields with the saved value after each save */}
      <label htmlFor="task-title" className="mt-4 block text-sm text-slate-300">
        Title
      </label>
      <input
        key={`title-${task.id}-${task.updated_at}`}
        id="task-title"
        defaultValue={task.title}
        disabled={!canWrite}
        maxLength={200}
        onBlur={(event) => {
          const title = event.target.value.trim()
          if (title === task.title) return
          if (!title) {
            event.target.value = task.title
            setError('The title cannot be empty.')
            return
          }
          void save({ title })
        }}
        className={fieldClass}
      />

      <div className="mt-4 grid grid-cols-2 gap-4">
        <div>
          <label htmlFor="task-status" className="block text-sm text-slate-300">
            Column
          </label>
          <select
            id="task-status"
            value={task.status_id}
            disabled={!canWrite}
            onChange={(event) => onMove(event.target.value)}
            className={fieldClass}
          >
            {statuses.map((status) => (
              <option key={status.id} value={status.id}>
                {status.name}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label htmlFor="task-priority" className="block text-sm text-slate-300">
            Priority
          </label>
          <select
            id="task-priority"
            value={task.priority}
            disabled={!canWrite}
            onChange={(event) => void save({ priority: event.target.value as Task['priority'] })}
            className={fieldClass}
          >
            {priorities.map((priority) => (
              <option key={priority} value={priority}>
                {priority}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label htmlFor="task-assignee" className="block text-sm text-slate-300">
            Assignee
          </label>
          <select
            id="task-assignee"
            value={task.assignee_id ?? ''}
            disabled={!canWrite}
            onChange={(event) => void save({ assignee_id: event.target.value || null })}
            className={fieldClass}
          >
            <option value="">Unassigned</option>
            {!assigneeIsKnown && <option value={task.assignee_id ?? ''}>Former member</option>}
            {members.map((member) => (
              <option key={member.user_id} value={member.user_id}>
                {member.name}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label htmlFor="task-due" className="block text-sm text-slate-300">
            Due date
          </label>
          <input
            id="task-due"
            type="date"
            value={task.due_date ?? ''}
            disabled={!canWrite}
            onChange={(event) => void save({ due_date: event.target.value || null })}
            className={fieldClass}
          />
        </div>
      </div>

      <label htmlFor="task-description" className="mt-4 block text-sm text-slate-300">
        Description
      </label>
      <textarea
        key={`description-${task.id}-${task.updated_at}`}
        id="task-description"
        rows={6}
        defaultValue={task.description}
        disabled={!canWrite}
        maxLength={10000}
        onBlur={(event) => {
          const description = event.target.value.trim()
          if (description !== task.description) void save({ description })
        }}
        className={fieldClass}
      />

      <p className="mt-4 flex items-center gap-2 text-xs text-slate-500">
        <PriorityBadge priority={task.priority} />
        Created {new Date(task.created_at).toLocaleDateString()}
      </p>
      <Comments
        workspaceId={workspaceId}
        projectId={projectId}
        taskNumber={task.number}
        role={role}
        myUserId={myUserId}
      />
      {canWrite && (
        <div className="mt-8 border-t border-slate-800 pt-6">
          {!confirmingDelete ? (
            <button
              onClick={() => setConfirmingDelete(true)}
              className="rounded-md bg-slate-800 px-4 py-2 text-sm text-red-300 hover:bg-slate-700"
            >
              Delete task…
            </button>
          ) : (
            <div className="flex items-center gap-3 text-sm">
              <button
                onClick={onDelete}
                disabled={remove.isPending}
                className="rounded-md bg-red-500 px-4 py-2 font-medium hover:bg-red-400 disabled:opacity-60"
              >
                Delete {task.identifier}
              </button>
              <button onClick={() => setConfirmingDelete(false)} className="text-slate-300 hover:text-white">
                Cancel
              </button>
            </div>
          )}
        </div>
      )}
    </aside>
  )
}