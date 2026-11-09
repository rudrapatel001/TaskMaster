import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { ApiError } from '../lib/api'
import { useUpdateProject } from '../lib/projects'
import type { Project } from '../lib/projects'
import { TextField } from './TextField'

const schema = z.object({
  name: z.string().trim().min(1, 'Name is required').max(100, 'Name must be at most 100 characters'),
  description: z.string().trim().max(2000, 'Description must be at most 2000 characters'),
})

type FormValues = z.infer<typeof schema>

/** Rename a project and edit its description. The key is shown but cannot be changed. */
export function ProjectSettingsForm({ workspaceId, project }: { workspaceId: string; project: Project }) {
  const update = useUpdateProject(workspaceId)
  const [message, setMessage] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: project.name, description: project.description },
  })

  async function onSubmit(values: FormValues) {
    setMessage(null)
    try {
      const saved = await update.mutateAsync({ id: project.id, ...values })
      reset({ name: saved.name, description: saved.description }) // the saved values become the new "unchanged"
      setMessage({ kind: 'ok', text: 'Saved.' })
    } catch (err) {
      if (err instanceof ApiError && err.status === 422) {
        for (const [field, text] of Object.entries(err.fields)) {
          if (field === 'name' || field === 'description') setError(field, { message: text })
        }
      } else if (err instanceof ApiError && err.status === 403) {
        setMessage({ kind: 'error', text: err.message })
      } else {
        setMessage({ kind: 'error', text: 'Could not save the changes. Please try again.' })
      }
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate className="rounded-xl bg-slate-800 p-6">
      <h2 className="font-semibold">Details</h2>

      <div className="mt-4">
        <label htmlFor="project-key" className="block text-sm text-slate-300">
          Key
        </label>
        <input
          id="project-key"
          value={project.key}
          disabled
          className="mt-1 w-32 rounded-md bg-slate-900 px-3 py-2 text-slate-400 ring-1 ring-slate-700"
        />
        <p className="mt-1 text-xs text-slate-500">
          The key is permanent: it starts every task id, like <strong>{project.key}</strong>-12.
        </p>
      </div>

      <TextField
        id="project-name"
        label="Name"
        error={errors.name?.message}
        {...register('name', { onChange: () => setMessage(null) })}
      />

      <div className="mt-4">
        <label htmlFor="project-description" className="block text-sm text-slate-300">
          Description
        </label>
        <textarea
          id="project-description"
          rows={4}
          {...register('description', { onChange: () => setMessage(null) })}
          className="mt-1 w-full rounded-md bg-slate-900 px-3 py-2 text-white outline-none ring-1 ring-slate-600 focus:ring-2 focus:ring-indigo-400"
        />
        {errors.description && <p className="mt-1 text-sm text-red-300">{errors.description.message}</p>}
      </div>

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