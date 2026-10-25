import { useState } from 'react'
import type { FormEvent } from 'react'
import { ApiError } from '../lib/api'
import { useCreateTask } from '../lib/tasks'

/**
 * "+ Add task" at the bottom of a column. It stays open after adding, so
 * several tasks can be typed in a row: Enter adds, Escape closes.
 */
export function AddTaskForm({
  workspaceId,
  projectId,
  statusId,
}: {
  workspaceId: string
  projectId: string
  statusId: string
}) {
  const create = useCreateTask(workspaceId, projectId)
  const [open, setOpen] = useState(false)
  const [title, setTitle] = useState('')
  const [error, setError] = useState<string | null>(null)

  function close() {
    setOpen(false)
    setTitle('')
    setError(null)
  }

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    const trimmed = title.trim()
    if (!trimmed) return
    if (trimmed.length > 200) {
      setError('Title must be at most 200 characters')
      return
    }

    setError(null)
    try {
      await create.mutateAsync({ title: trimmed, status_id: statusId })
      setTitle('')
    } catch (err) {
      setError(err instanceof ApiError && err.status === 403 ? err.message : 'Could not add the task. Please try again.')
    }
  }

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className="mt-2 w-full rounded-md px-2 py-1.5 text-left text-sm text-slate-400 hover:bg-slate-800 hover:text-white"
      >
        + Add task
      </button>
    )
  }

  return (
    <form onSubmit={onSubmit} className="mt-2">
      <label htmlFor={`new-task-${statusId}`} className="sr-only">
        Task title
      </label>
      <input
        id={`new-task-${statusId}`}
        autoFocus
        value={title}
        onChange={(event) => setTitle(event.target.value)}
        onKeyDown={(event) => event.key === 'Escape' && close()}
        placeholder="What needs doing?"
        autoComplete="off"
        className="w-full rounded-md bg-slate-900 px-3 py-2 text-sm text-white outline-none ring-1 ring-slate-600 focus:ring-2 focus:ring-indigo-400"
      />
      {error && (
        <p role="alert" className="mt-1 text-xs text-red-300">
          {error}
        </p>
      )}
      <div className="mt-2 flex items-center gap-3 text-sm">
        <button
          type="submit"
          disabled={create.isPending || !title.trim()}
          className="rounded-md bg-indigo-500 px-3 py-1 font-medium hover:bg-indigo-400 disabled:opacity-50"
        >
          Add
        </button>
        <button type="button" onClick={close} className="text-slate-400 hover:text-white">
          Cancel
        </button>
      </div>
    </form>
  )
}