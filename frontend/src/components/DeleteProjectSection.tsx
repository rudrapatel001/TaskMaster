import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ApiError } from '../lib/api'
import { useDeleteProject } from '../lib/projects'
import type { Project } from '../lib/projects'

/** Deleting a project: it asks you to type the project's key first. */
export function DeleteProjectSection({
  workspaceId,
  workspaceSlug,
  project,
}: {
  workspaceId: string
  workspaceSlug: string
  project: Project
}) {
  const remove = useDeleteProject(workspaceId)
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [typed, setTyped] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function onDelete() {
    setError(null)
    try {
      await remove.mutateAsync(project.id)
      navigate(`/w/${workspaceSlug}/projects`, { replace: true })
    } catch (err) {
      setError(err instanceof ApiError && err.status === 403 ? err.message : 'Could not delete the project. Please try again.')
    }
  }

  return (
    <section className="rounded-xl border border-red-500/40 p-6">
      <h2 className="font-semibold text-red-300">Delete project</h2>
      <p className="mt-1 text-sm text-slate-400">
        This permanently deletes <strong>{project.name}</strong> with all of its tasks, comments and columns, for
        everyone. It cannot be undone.
      </p>

      {error && (
        <p role="alert" className="mt-3 rounded-md bg-red-500/10 px-3 py-2 text-sm text-red-300">
          {error}
        </p>
      )}

      {!open ? (
        <button
          onClick={() => setOpen(true)}
          className="mt-4 rounded-md bg-slate-700 px-4 py-2 text-sm hover:bg-slate-600"
        >
          Delete project…
        </button>
      ) : (
        <div className="mt-4">
          <label htmlFor="confirm-project-key" className="block text-sm text-slate-300">
            Type <strong>{project.key}</strong> to confirm
          </label>
          <input
            id="confirm-project-key"
            value={typed}
            onChange={(event) => setTyped(event.target.value)}
            autoComplete="off"
            className="mt-1 w-full rounded-md bg-slate-900 px-3 py-2 text-white outline-none ring-1 ring-slate-600 focus:ring-2 focus:ring-red-400"
          />
          <div className="mt-3 flex items-center gap-3">
            <button
              onClick={onDelete}
              disabled={typed !== project.key || remove.isPending}
              className="rounded-md bg-red-500 px-4 py-2 text-sm font-medium hover:bg-red-400 disabled:opacity-40"
            >
              Delete forever
            </button>
            <button
              onClick={() => {
                setOpen(false)
                setTyped('')
              }}
              className="text-sm text-slate-300 hover:text-white"
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </section>
  )
}