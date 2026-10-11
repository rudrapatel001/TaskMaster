import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { TextField } from '../components/TextField'
import { api, ApiError } from '../lib/api'
import type { User } from '../lib/api'

// These rules mirror the backend's. They give instant feedback, but the server
// always has the final say.
const schema = z.object({
  name: z.string().trim().min(1, 'Name is required').max(100, 'Name must be at most 100 characters'),
  email: z.string().trim().min(1, 'Email is required').email('Enter a valid email address'),
  password: z
    .string()
    .min(1, 'Password is required')
    .min(8, 'Password must be at least 8 characters')
    // The backend's limit is 72 bytes, and non-English characters use more than one.
    .refine((value) => new TextEncoder().encode(value).length <= 72, 'Password is too long (72 bytes maximum)'),
})

type FormValues = z.infer<typeof schema>

export function RegisterPage() {
  const { state, login } = useAuth()
  const navigate = useNavigate()
  const [formError, setFormError] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema) })

  if (state.status === 'authenticated') {
    return <Navigate to="/" replace />
  }

  async function onSubmit(values: FormValues) {
    setFormError(null)

    try {
      await api<User>('/auth/register', { method: 'POST', body: values, auth: false })
    } catch (err) {
      if (!(err instanceof ApiError)) {
        setFormError('Could not reach the server. Is the backend running?')
      } else if (err.status === 409) {
        setError('email', { message: 'An account with this email already exists' })
      } else if (err.status === 422) {
        for (const [field, message] of Object.entries(err.fields)) {
          if (field === 'name' || field === 'email' || field === 'password') setError(field, { message })
        }
      } else if (err.status === 429) {
        setFormError('Too many attempts. Please wait a minute and try again.')
      } else {
        setFormError('Something went wrong. Please try again.')
      }
      return
    }

    // The account exists now. Log in with the same details.
    try {
      await login(values.email, values.password)
      navigate('/', { replace: true })
    } catch {
      navigate('/login', { replace: true, state: { notice: 'Account created. Please log in.' } })
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-900 px-4">
      <form
        onSubmit={handleSubmit(onSubmit)}
        noValidate
        className="w-full max-w-sm rounded-xl bg-slate-800 p-8 shadow-lg"
      >
        <h1 className="text-2xl font-bold text-white">Create your account</h1>

        {formError && (
          <p role="alert" className="mt-4 rounded-md bg-red-500/10 px-3 py-2 text-sm text-red-300">
            {formError}
          </p>
        )}

        <TextField
          id="name"
          label="Name"
          autoComplete="name"
          error={errors.name?.message}
          {...register('name')}
        />
        <TextField
          id="email"
          label="Email"
          type="email"
          autoComplete="email"
          error={errors.email?.message}
          {...register('email')}
        />
        <TextField
          id="password"
          label="Password"
          type="password"
          autoComplete="new-password"
          error={errors.password?.message}
          {...register('password')}
        />
        <p className="mt-1 text-xs text-slate-500">At least 8 characters.</p>

        <button
          type="submit"
          disabled={isSubmitting}
          className="mt-6 w-full rounded-md bg-indigo-500 px-4 py-2 font-medium text-white hover:bg-indigo-400 disabled:opacity-60"
        >
          {isSubmitting ? 'Creating account…' : 'Create account'}
        </button>

        <p className="mt-6 text-center text-sm text-slate-400">
          Already have an account?{' '}
          <Link to="/login" className="text-indigo-300 hover:underline">
            Log in
          </Link>
        </p>
      </form>
    </div>
  )
}