import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useNavigate } from 'react-router-dom'
import { TextField } from '../components/TextField'
import { ApiError } from '../lib/api'
import { useCreateWorkspace, useWorkspaces } from '../lib/workspaces'

const schema = z.object({
  name: z.string().trim().min(1, 'Name is required').max(60, 'Name must be at most 60 characters'),
})

type FormValues = z.infer<typeof schema>

export function CreateWorkspacePage() {
  const navigate = useNavigate()
  const { data: workspaces } = useWorkspaces()
  const create = useCreateWorkspace()
  const [formError, setFormError] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema) })

  const isFirst = workspaces?.length === 0

  async function onSubmit(values: FormValues) {
    setFormError(null)
    try {
      const created = await create.mutateAsync(values.name)
      navigate(`/w/${created.slug}`)
    } catch (err) {
      if (err instanceof ApiError && err.status === 422 && err.fields.name) {
        setError('name', { message: err.fields.name })
      } else {
        setFormError('Could not create the workspace. Please try again.')
      }
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate className="mx-auto max-w-md rounded-xl bg-slate-800 p-8">
      <h1 className="text-2xl font-bold">{isFirst ? 'Create your first workspace' : 'Create a workspace'}</h1>
      <p className="mt-2 text-sm text-slate-400">
        A workspace is a shared space for a team. You will be its owner, and you can invite people later.
      </p>

      {formError && (
        <p role="alert" className="mt-4 rounded-md bg-red-500/10 px-3 py-2 text-sm text-red-300">
          {formError}
        </p>
      )}

      <TextField id="name" label="Workspace name" error={errors.name?.message} {...register('name')} />

      <button
        type="submit"
        disabled={isSubmitting}
        className="mt-6 w-full rounded-md bg-indigo-500 px-4 py-2 font-medium hover:bg-indigo-400 disabled:opacity-60"
      >
        {isSubmitting ? 'Creating…' : 'Create workspace'}
      </button>
    </form>
  )
}