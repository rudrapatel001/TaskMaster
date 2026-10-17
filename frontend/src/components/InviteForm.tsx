import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { ApiError } from '../lib/api'
import { useCreateInvitation } from '../lib/members'
import type { CreatedInvitation } from '../lib/members'
import type { InviteRole } from '../lib/permissions'
import { TextField } from './TextField'

const schema = z.object({
  email: z.string().trim().min(1, 'Email is required').email('Enter a valid email address'),
  role: z.enum(['viewer', 'member', 'admin']),
})

type FormValues = z.infer<typeof schema>

/** Invite someone by email. Shows the one-time link afterwards, ready to copy. */
export function InviteForm({ workspaceId, roles }: { workspaceId: string; roles: InviteRole[] }) {
  const create = useCreateInvitation(workspaceId)
  const [formError, setFormError] = useState<string | null>(null)
  const [created, setCreated] = useState<CreatedInvitation | null>(null)
  const [copied, setCopied] = useState(false)

  const defaultRole: InviteRole = roles.includes('member') ? 'member' : roles[0]

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: '', role: defaultRole },
  })

  async function onSubmit(values: FormValues) {
    setFormError(null)
    setCopied(false)
    try {
      setCreated(await create.mutateAsync(values))
      reset({ email: '', role: values.role })
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setError('email', { message: 'This person is already a member' })
      } else if (err instanceof ApiError && err.status === 422) {
        for (const [field, message] of Object.entries(err.fields)) {
          if (field === 'email' || field === 'role') setError(field, { message })
        }
      } else if (err instanceof ApiError && err.status === 403) {
        setFormError(err.message)
      } else {
        setFormError('Could not create the invitation. Please try again.')
      }
    }
  }

  const link = created ? `${window.location.origin}/invite/${created.token}` : ''

  async function copy() {
    try {
      await navigator.clipboard.writeText(link)
      setCopied(true)
    } catch {
      setCopied(false) // the link is also selectable in the box, so it can be copied by hand
    }
  }

  return (
    <div className="rounded-xl bg-slate-800 p-6">
      <h2 className="font-semibold">Invite someone</h2>

      {created && (
        <div role="status" className="mt-4 rounded-md bg-emerald-500/10 p-4 text-sm">
          <p className="text-emerald-300">
            Invitation created for <strong>{created.email}</strong>. Send them this link. It works once, only
            for that email address, and is shown <strong>only now</strong>.
          </p>
          <div className="mt-3 flex gap-2">
            <input
              readOnly
              aria-label="Invitation link"
              value={link}
              onFocus={(event) => event.currentTarget.select()}
              className="min-w-0 flex-1 rounded-md bg-slate-900 px-3 py-2 text-slate-200 ring-1 ring-slate-600"
            />
            <button onClick={copy} className="rounded-md bg-indigo-500 px-3 py-2 font-medium hover:bg-indigo-400">
              {copied ? 'Copied!' : 'Copy link'}
            </button>
          </div>
        </div>
      )}

      <form onSubmit={handleSubmit(onSubmit)} noValidate>
        {formError && (
          <p role="alert" className="mt-4 rounded-md bg-red-500/10 px-3 py-2 text-sm text-red-300">
            {formError}
          </p>
        )}

        <TextField
          id="invite-email"
          label="Email address"
          type="email"
          autoComplete="off"
          error={errors.email?.message}
          {...register('email')}
        />

        <div className="mt-4">
          <label htmlFor="invite-role" className="block text-sm text-slate-300">
            Role
          </label>
          <select
            id="invite-role"
            {...register('role')}
            className="mt-1 rounded-md bg-slate-900 px-3 py-2 text-white ring-1 ring-slate-600 focus:ring-2 focus:ring-indigo-400"
          >
            {roles.map((role) => (
              <option key={role} value={role}>
                {role}
              </option>
            ))}
          </select>
        </div>

        <button
          type="submit"
          disabled={isSubmitting}
          className="mt-4 rounded-md bg-indigo-500 px-4 py-2 text-sm font-medium hover:bg-indigo-400 disabled:opacity-60"
        >
          {isSubmitting ? 'Creating…' : 'Create invitation'}
        </button>
      </form>
    </div>
  )
}