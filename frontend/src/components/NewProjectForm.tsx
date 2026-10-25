import { useEffect, useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useNavigate } from 'react-router-dom'
import { ApiError } from '../lib/api'
import { suggestKey, useCreateProject } from '../lib/projects'
import { TextField } from './TextField'

const schema = z.object({
  name: z.string().trim().min(1, 'Name is required').max(100, 'Name must be at most 100 characters'),
  key: z
    .string()
    .trim()
    .regex(/^[A-Za-z][A-Za-z0-9]{1,9}$/, 'Use 2 to 10 letters or digits, starting with a letter'),
  description: z.string().trim().max(2000, 'Description must be at most 2000 characters'),
})

type FormValues = z.infer<typeof schema>

/** Create a project. The key is suggested from the name until you type your own. */
export function NewProjectForm({
  workspaceId,
  workspaceSlug,
  onCancel,
}: {
  workspaceId: string
  workspaceSlug: string
  onCancel: () => void
}) {
  const create = useCreateProject(workspaceId)
  const navigate = useNavigate()
  const [formError, setFormError] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    control,
    setValue,
    setError,
    formState: { errors, isSubmitting, dirtyFields },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: '', key: '', description: '' },
  })

  const name = useWatch({ control, name: 'name' })
  const keyWasEdited = !!dirtyFields.key
  useEffect(() => {
    if (!keyWasEdited) setValue('key', suggestKey(name))
  }, [name, keyWasEdited, setValue])

  async function onSubmit(values: FormValues) {
    setFormError(null)
    try {
      const created = await create.mutateAsync({ ...values, key: values.key.toUpperCase() })
      navigate(`/w/${workspaceSlug}/projects/${created.key}`)
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setError('key', { message: 'This key is already used in this workspace' })
      } else if (err instanceof ApiError && err.status === 422) {
        for (const [field, message] of Object.entries(err.fields)) {
          if (field === 'name' || field === 'key' || field === 'description') setError(field, { message })
        }
      } else {
        setFormError('Could not create the project. Please try again.')
      }
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate className="rounded-xl bg-slate-800 p-6">
      <h2 className="font-semibold">New project</h2>

      {formError && (
        <p role="alert" className="mt-3 rounded-md bg-red-500/10 px-3 py-2 text-sm text-red-300">
          {formError}
        </p>
      )}

      <TextField id="project-name" label="Name" autoFocus error={errors.name?.message} {...register('name')} />
      <TextField
        id="project-key"
        label="Key"
        autoComplete="off"
        error={errors.key?.message}
        {...register('key')}
      />
      <p className="mt-1 text-xs text-slate-500">
        Short and permanent: it starts every task id, like <strong>WEB</strong>-12.
      </p>

      <div className="mt-4">
        <label htmlFor="project-description" className="block text-sm text-slate-300">
          Description (optional)
        </label>
        <textarea
          id="project-description"
          rows={3}
          {...register('description')}
          className="mt-1 w-full rounded-md bg-slate-900 px-3 py-2 text-white outline-none ring-1 ring-slate-600 focus:ring-2 focus:ring-indigo-400"
        />
        {errors.description && <p className="mt-1 text-sm text-red-300">{errors.description.message}</p>}
      </div>

      <div className="mt-4 flex items-center gap-3">
        <button
          type="submit"
          disabled={isSubmitting}
          className="rounded-md bg-indigo-500 px-4 py-2 text-sm font-medium hover:bg-indigo-400 disabled:opacity-60"
        >
          {isSubmitting ? 'Creating…' : 'Create project'}
        </button>
        <button type="button" onClick={onCancel} className="text-sm text-slate-300 hover:text-white">
          Cancel
        </button>
      </div>
    </form>
  )
}