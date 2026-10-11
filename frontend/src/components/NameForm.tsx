import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { api, ApiError } from '../lib/api'
import type { User } from '../lib/api'
import { TextField } from './TextField'

const schema = z.object({
  name: z.string().trim().min(1, 'Name is required').max(100, 'Name must be at most 100 characters'),
})

type FormValues = z.infer<typeof schema>

/** Lets the logged-in user change their display name. */
export function NameForm({ user, onSaved }: { user: User; onSaved: (user: User) => void }) {
  const [message, setMessage] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null)

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: user.name },
  })

  async function onSubmit(values: FormValues) {
    setMessage(null)
    try {
      const updated = await api<User>('/me', { method: 'PATCH', body: values })
      onSaved(updated)
      reset({ name: updated.name }) // the saved value is the new starting point
      setMessage({ kind: 'ok', text: 'Saved.' })
    } catch (err) {
      if (err instanceof ApiError && err.status === 422 && err.fields.name) {
        setError('name', { message: err.fields.name })
      } else {
        setMessage({ kind: 'error', text: 'Could not save your name. Please try again.' })
      }
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate className="mt-10 rounded-xl bg-slate-800 p-6">
      <h2 className="font-semibold">Your profile</h2>

      <TextField
        id="profile-name"
        label="Display name"
        autoComplete="name"
        error={errors.name?.message}
        {...register('name', { onChange: () => setMessage(null) })}
      />

      <div className="mt-4 flex items-center gap-4">
        <button
          type="submit"
          disabled={isSubmitting || !isDirty}
          className="rounded-md bg-indigo-500 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-400 disabled:opacity-50"
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