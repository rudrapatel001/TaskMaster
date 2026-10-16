import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useNavigate } from 'react-router-dom'
import { ApiError } from '../lib/api'
import { can } from '../lib/permissions'
import { useDeleteWorkspace, useLeaveWorkspace, useRenameWorkspace } from '../lib/workspaces'
import type { Workspace } from '../lib/workspaces'
import { TextField } from './TextField'

/** What the server said, if it said something we can show; otherwise a general message. */
function messageFor(err: unknown, fallback: string): string {
  if (err instanceof ApiError && (err.status === 403 || err.status === 409)) return err.message
  return fallback
}

/** Settings for one workspace. Each section only appears for roles that may use it. */
export function WorkspaceSettings({ workspace }: { workspace: Workspace }) {
  return (
    <div className="mt-10 space-y-6">
      {can(workspace.role, 'workspace:update') && <RenameForm workspace={workspace} />}
      <LeaveSection workspace={workspace} />
      {can(workspace.role, 'workspace:delete') && <DeleteSection workspace={workspace} />}
    </div>
  )
}

const renameSchema = z.object({
  name: z.string().trim().min(1, 'Name is required').max(60, 'Name must be at most 60 characters'),
})

function RenameForm({ workspace }: { workspace: Workspace }) {
  const rename = useRenameWorkspace()
  const [message, setMessage] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<z.infer<typeof renameSchema>>({
    resolver: zodResolver(renameSchema),
    defaultValues: { name: workspace.name },
  })

  async function onSubmit(values: z.infer<typeof renameSchema>) {
    setMessage(null)
    try {
      const updated = await rename.mutateAsync({ id: workspace.id, name: values.name })
      reset({ name: updated.name })
      setMessage({ kind: 'ok', text: 'Saved.' })
    } catch (err) {
      if (err instanceof ApiError && err.status === 422 && err.fields.name) {
        setError('name', { message: err.fields.name })
      } else {
        setMessage({ kind: 'error', text: messageFor(err, 'Could not rename the workspace.') })
      }
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate className="rounded-xl bg-slate-800 p-6">
      <h2 className="font-semibold">Workspace name</h2>
      <TextField
        id="workspace-name"
        label="Name"
        error={errors.name?.message}
        {...register('name', { onChange: () => setMessage(null) })}
      />
      <div className="mt-4 flex items-center gap-4">
        <button
          type="submit"
          disabled={isSubmitting || !isDirty}
          className="rounded-md bg-indigo-500 px-4 py-2 text-sm font-medium hover:bg-indigo-400 disabled:opacity-50"
        >
          {isSubmitting ? 'Saving…' : 'Save'}
        </button>
        {message && (
          <p
            role={message.kind === 'error' ? 'alert' : 'status'}
            className={message.kind === 'ok' ? 'text-sm text-emerald-300' : 'text-sm text-red-300'}
          >
            {message.text}
          </p>
        )}
      </div>
    </form>
  )
}

function LeaveSection({ workspace }: { workspace: Workspace }) {
  const leave = useLeaveWorkspace()
  const navigate = useNavigate()
  const [confirming, setConfirming] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function onLeave() {
    setError(null)
    try {
      await leave.mutateAsync(workspace.id)
      navigate('/', { replace: true })
    } catch (err) {
      setError(messageFor(err, 'Could not leave the workspace.'))
    }
  }

  return (
    <section className="rounded-xl bg-slate-800 p-6">
      <h2 className="font-semibold">Leave workspace</h2>
      <p className="mt-1 text-sm text-slate-400">You will lose access until someone invites you again.</p>

      {error && (
        <p role="alert" className="mt-3 rounded-md bg-red-500/10 px-3 py-2 text-sm text-red-300">
          {error}
        </p>
      )}

      {!confirming ? (
        <button
          onClick={() => setConfirming(true)}
          className="mt-4 rounded-md bg-slate-700 px-4 py-2 text-sm hover:bg-slate-600"
        >
          Leave workspace…
        </button>
      ) : (
        <div className="mt-4 flex items-center gap-3">
          <button
            onClick={onLeave}
            disabled={leave.isPending}
            className="rounded-md bg-red-500 px-4 py-2 text-sm font-medium hover:bg-red-400 disabled:opacity-60"
          >
            Yes, leave {workspace.name}
          </button>
          <button onClick={() => setConfirming(false)} className="text-sm text-slate-300 hover:text-white">
            Cancel
          </button>
        </div>
      )}
    </section>
  )
}

function DeleteSection({ workspace }: { workspace: Workspace }) {
  const remove = useDeleteWorkspace()
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [typed, setTyped] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function onDelete() {
    setError(null)
    try {
      await remove.mutateAsync(workspace.id)
      navigate('/', { replace: true })
    } catch (err) {
      setError(messageFor(err, 'Could not delete the workspace.'))
    }
  }

  return (
    <section className="rounded-xl border border-red-500/40 p-6">
      <h2 className="font-semibold text-red-300">Delete workspace</h2>
      <p className="mt-1 text-sm text-slate-400">
        This permanently deletes the workspace and everything in it for all members. It cannot be undone.
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
          Delete workspace…
        </button>
      ) : (
        <div className="mt-4">
          <label htmlFor="confirm-name" className="block text-sm text-slate-300">
            Type <strong>{workspace.name}</strong> to confirm
          </label>
          <input
            id="confirm-name"
            value={typed}
            onChange={(event) => setTyped(event.target.value)}
            autoComplete="off"
            className="mt-1 w-full rounded-md bg-slate-900 px-3 py-2 text-white outline-none ring-1 ring-slate-600 focus:ring-2 focus:ring-red-400"
          />
          <div className="mt-3 flex items-center gap-3">
            <button
              onClick={onDelete}
              disabled={typed !== workspace.name || remove.isPending}
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