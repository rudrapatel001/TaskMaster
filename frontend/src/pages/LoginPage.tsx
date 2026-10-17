import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { TextField } from '../components/TextField'
import { ApiError } from '../lib/api'

// Login only checks that something was typed. Password rules are never shown
// here: they would tell an attacker what a valid password looks like.
const schema = z.object({
  email: z.string().trim().min(1, 'Email is required').email('Enter a valid email address'),
  password: z.string().min(1, 'Password is required'),
})

type FormValues = z.infer<typeof schema>

interface LocationState {
  from?: { pathname: string }
  notice?: string
}

export function LoginPage() {
  const { state, login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const locationState = location.state as LocationState | null
  const [formError, setFormError] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema) })

  // Already logged in? Nothing to do here. This also fires right after a
  // successful login, so it must send you where you were going, not just home.
  if (state.status === 'authenticated') {
    return <Navigate to={locationState?.from?.pathname ?? '/'} replace />
  }

  async function onSubmit(values: FormValues) {
    setFormError(null)
    try {
      await login(values.email, values.password)
      navigate(locationState?.from?.pathname ?? '/', { replace: true })
    } catch (err) {
      if (!(err instanceof ApiError)) {
        setFormError('Could not reach the server. Is the backend running?')
      } else if (err.status === 401) {
        setFormError('Invalid email or password.')
      } else if (err.status === 422) {
        // The server named the fields that are wrong: show each next to its input.
        for (const [field, message] of Object.entries(err.fields)) {
          if (field === 'email' || field === 'password') setError(field, { message })
        }
      } else if (err.status === 429) {
        setFormError('Too many attempts. Please wait a minute and try again.')
      } else {
        setFormError('Something went wrong. Please try again.')
      }
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-900 px-4">
      <form
        onSubmit={handleSubmit(onSubmit)}
        noValidate
        className="w-full max-w-sm rounded-xl bg-slate-800 p-8 shadow-lg"
      >
        <h1 className="text-2xl font-bold text-white">Log in to TaskMaster</h1>

        {locationState?.notice && (
          <p role="status" className="mt-4 rounded-md bg-emerald-500/10 px-3 py-2 text-sm text-emerald-300">
            {locationState.notice}
          </p>
        )}

        {formError && (
          <p role="alert" className="mt-4 rounded-md bg-red-500/10 px-3 py-2 text-sm text-red-300">
            {formError}
          </p>
        )}

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
          autoComplete="current-password"
          error={errors.password?.message}
          {...register('password')}
        />

        <button
          type="submit"
          disabled={isSubmitting}
          className="mt-6 w-full rounded-md bg-indigo-500 px-4 py-2 font-medium text-white hover:bg-indigo-400 disabled:opacity-60"
        >
          {isSubmitting ? 'Logging in…' : 'Log in'}
        </button>

        <p className="mt-6 text-center text-sm text-slate-400">
          New here?{' '}
          <Link to="/register" state={locationState} className="text-indigo-300 hover:underline">
            Create an account
          </Link>
        </p>
      </form>
    </div>
  )
}