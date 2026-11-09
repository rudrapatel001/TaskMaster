import { useState } from 'react'
import type { FormEvent } from 'react'
import { ApiError } from '../lib/api'
import {
  useCreateStatus,
  useDeleteStatus,
  useMoveStatus,
  useProject,
  useUpdateStatus,
} from '../lib/projects'
import type { Status, StatusCategory } from '../lib/projects'
import { QueryError } from './QueryError'

const MAX_COLUMNS = 12

const categoryLabels: Record<StatusCategory, string> = {
  todo: 'To do',
  in_progress: 'In progress',
  done: 'Done',
}
const categories = Object.keys(categoryLabels) as StatusCategory[]

const fieldClass =
  'rounded-md bg-slate-800 px-3 py-2 text-sm text-white outline-none ring-1 ring-slate-600 focus:ring-2 focus:ring-indigo-400'

/** The server's reason when it gave one we can show; otherwise a general message. */
function reasonFor(err: unknown): string {
  if (err instanceof ApiError) {
    const firstFieldError = Object.values(err.fields)[0]
    if (err.status === 403 || err.status === 409) return err.message
    if (err.status === 422 && firstFieldError) return firstFieldError
  }
  return 'That did not work. Please try again.'
}

/** Add, rename, reorder and delete the columns of a project's board. */
export function ColumnsSettings({ workspaceId, projectId }: { workspaceId: string; projectId: string }) {
  const project = useProject(workspaceId, projectId)
  const create = useCreateStatus(workspaceId, projectId)
  const update = useUpdateStatus(workspaceId, projectId)
  const move = useMoveStatus(workspaceId, projectId)
  const remove = useDeleteStatus(workspaceId, projectId)

  const [error, setError] = useState<string | null>(null)
  const [newName, setNewName] = useState('')
  const [newCategory, setNewCategory] = useState<StatusCategory>('in_progress')

  /** Runs a change, shows the reason if it fails, and says whether it worked. */
  async function run(action: () => Promise<unknown>): Promise<boolean> {
    setError(null)
    try {
      await action()
      return true
    } catch (err) {
      setError(reasonFor(err))
      return false
    }
  }

  if (project.isPending) return <p className="text-slate-400">Loading the columns…</p>
  if (project.isError) {
    return <QueryError message="Could not load the columns." onRetry={() => void project.refetch()} />
  }

  const statuses = project.data.statuses
  const full = statuses.length >= MAX_COLUMNS

  async function onAdd(event: FormEvent) {
    event.preventDefault()
    const name = newName.trim()
    if (!name || full) return
    if (await run(() => create.mutateAsync({ name, category: newCategory }))) setNewName('')
  }

  return (
    <section className="rounded-xl bg-slate-800 p-6">
      <h2 className="font-semibold">Board columns</h2>
      <p className="mt-1 text-sm text-slate-400">
        Columns appear on the board from left to right in this order. The category tells the app what a column
        means, whatever you call it: for example, a task in a "Done" column is never shown as overdue.
      </p>

      {error && (
        <p role="alert" className="mt-3 rounded-md bg-red-500/10 px-3 py-2 text-sm text-red-300">
          {error}
        </p>
      )}

      <ul className="mt-4 space-y-2">
        {statuses.map((status, index) => (
          <ColumnRow
            key={status.id}
            status={status}
            isFirst={index === 0}
            isLast={index === statuses.length - 1}
            others={statuses.filter((other) => other.id !== status.id)}
            onRename={(name) => run(() => update.mutateAsync({ id: status.id, name }))}
            onCategory={(category) => run(() => update.mutateAsync({ id: status.id, category }))}
            // "Up" means: go directly before the column that is now before this one.
            onMoveUp={() => run(() => move.mutateAsync({ id: status.id, before: statuses[index - 1].id }))}
            // "Down" means: go directly before the column that comes two places after, or last.
            onMoveDown={() =>
              run(() => move.mutateAsync({ id: status.id, before: statuses[index + 2]?.id ?? null }))
            }
            onDelete={(moveTo) => run(() => remove.mutateAsync({ id: status.id, moveTo }))}
          />
        ))}
      </ul>

      <form onSubmit={onAdd} className="mt-5 flex flex-wrap items-center gap-2">
        <label htmlFor="new-column-name" className="sr-only">
          New column name
        </label>
        <input
          id="new-column-name"
          value={newName}
          onChange={(event) => setNewName(event.target.value)}
          placeholder="New column, e.g. Blocked"
          maxLength={50}
          autoComplete="off"
          disabled={full}
          className={`${fieldClass} min-w-0 flex-1`}
        />
        <label htmlFor="new-column-category" className="sr-only">
          Category of the new column
        </label>
        <select
          id="new-column-category"
          value={newCategory}
          onChange={(event) => setNewCategory(event.target.value as StatusCategory)}
          disabled={full}
          className={fieldClass}
        >
          {categories.map((category) => (
            <option key={category} value={category}>
              {categoryLabels[category]}
            </option>
          ))}
        </select>
        <button
          type="submit"
          disabled={full || create.isPending || !newName.trim()}
          className="rounded-md bg-indigo-500 px-4 py-2 text-sm font-medium hover:bg-indigo-400 disabled:opacity-50"
        >
          Add column
        </button>
      </form>
      {full && <p className="mt-2 text-xs text-slate-500">A project can have at most {MAX_COLUMNS} columns.</p>}
    </section>
  )
}

function ColumnRow({
  status,
  isFirst,
  isLast,
  others,
  onRename,
  onCategory,
  onMoveUp,
  onMoveDown,
  onDelete,
}: {
  status: Status
  isFirst: boolean
  isLast: boolean
  others: Status[]
  onRename: (name: string) => Promise<boolean>
  onCategory: (category: StatusCategory) => Promise<boolean>
  onMoveUp: () => Promise<boolean>
  onMoveDown: () => Promise<boolean>
  onDelete: (moveTo: string) => Promise<boolean>
}) {
  const [deleting, setDeleting] = useState(false)
  const [target, setTarget] = useState('')

  // The chosen column, or the first other one if that choice no longer exists.
  const selectedTarget = others.some((other) => other.id === target) ? target : (others[0]?.id ?? '')
  const onlyColumn = others.length === 0

  return (
    <li className="rounded-lg bg-slate-900 p-3">
      <div className="flex flex-wrap items-center gap-2">
        <span className="flex flex-col">
          <button
            onClick={() => void onMoveUp()}
            disabled={isFirst}
            aria-label={`Move ${status.name} up`}
            className="rounded px-1.5 text-xs leading-5 text-slate-400 hover:bg-slate-700 hover:text-white disabled:opacity-30"
          >
            ▲
          </button>
          <button
            onClick={() => void onMoveDown()}
            disabled={isLast}
            aria-label={`Move ${status.name} down`}
            className="rounded px-1.5 text-xs leading-5 text-slate-400 hover:bg-slate-700 hover:text-white disabled:opacity-30"
          >
            ▼
          </button>
        </span>

        {/* The key restarts the box with the saved name after a save (or a rejected rename). */}
        <input
          key={`name-${status.id}-${status.name}`}
          aria-label={`Name of the ${status.name} column`}
          defaultValue={status.name}
          maxLength={50}
          onBlur={(event) => {
            const input = event.target
            const name = input.value.trim()
            if (name === status.name) return
            if (!name) {
              input.value = status.name
              return
            }
            void onRename(name).then((worked) => {
              if (!worked) input.value = status.name
            })
          }}
          onKeyDown={(event) => {
            if (event.key === 'Enter') event.currentTarget.blur()
          }}
          className={`${fieldClass} min-w-0 flex-1`}
        />

        <select
          aria-label={`Category of the ${status.name} column`}
          value={status.category}
          onChange={(event) => void onCategory(event.target.value as StatusCategory)}
          className={fieldClass}
        >
          {categories.map((category) => (
            <option key={category} value={category}>
              {categoryLabels[category]}
            </option>
          ))}
        </select>

        <button
          onClick={() => setDeleting((open) => !open)}
          disabled={onlyColumn}
          title={onlyColumn ? 'A project needs at least one column' : undefined}
          aria-label={`Delete the ${status.name} column`}
          className="rounded-md px-3 py-2 text-sm text-red-300 hover:bg-slate-800 disabled:opacity-40"
        >
          Delete…
        </button>
      </div>

      {deleting && !onlyColumn && (
        <div className="mt-3 rounded-md bg-slate-800 p-3 text-sm">
          <label htmlFor={`move-to-${status.id}`} className="block text-slate-300">
            Move any tasks in <strong>{status.name}</strong> to:
          </label>
          <select
            id={`move-to-${status.id}`}
            value={selectedTarget}
            onChange={(event) => setTarget(event.target.value)}
            className={`${fieldClass} mt-1`}
          >
            {others.map((other) => (
              <option key={other.id} value={other.id}>
                {other.name}
              </option>
            ))}
          </select>
          <div className="mt-3 flex items-center gap-3">
            <button
              onClick={() => void onDelete(selectedTarget)}
              disabled={!selectedTarget}
              className="rounded-md bg-red-500 px-4 py-2 font-medium hover:bg-red-400 disabled:opacity-50"
            >
              Delete column
            </button>
            <button onClick={() => setDeleting(false)} className="text-slate-300 hover:text-white">
              Cancel
            </button>
          </div>
        </div>
      )}
    </li>
  )
}