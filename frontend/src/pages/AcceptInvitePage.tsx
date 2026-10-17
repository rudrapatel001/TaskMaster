import { useState } from 'react'
import type { ReactNode } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { ApiError } from '../lib/api'
import { useAcceptInvitation, useInvitationPreview } from '../lib/members'

/** The page an invitation link opens. It works logged out (it only shows a preview) and logged in. */
export function AcceptInvitePage() {
  const { token = '' } = useParams()
  const { state, logout } = useAuth()
  const location = useLocation()
  const navigate = useNavigate()
  const preview = useInvitationPreview(token)
  const accept = useAcceptInvitation()
  const [error, setError] = useState<string | null>(null)

  if (preview.isPending || state.status === 'loading') {
    return <Card><p className="text-slate-400">Loading…</p></Card>
  }

  if (preview.isError) {
    const gone = preview.error instanceof ApiError && preview.error.status === 404
    return (
      <Card>
        <h1 className="text-xl font-bold">{gone ? 'This invitation link is no longer valid' : 'Something went wrong'}</h1>
        <p className="mt-2 text-sm text-slate-400">
          {gone
            ? 'It may have expired, been used already, or been cancelled. Ask the person who invited you for a new one.'
            : 'We could not check this invitation. Please try again.'}
        </p>
        <Link to="/" className="mt-4 inline-block text-indigo-300 hover:underline">
          Go to TaskMaster
        </Link>
      </Card>
    )
  }

  const invitation = preview.data
  const who = invitation.invited_by || 'Someone'

  async function onAccept() {
    setError(null)
    try {
      const joined = await accept.mutateAsync(token)
      navigate(`/w/${joined.slug}`, { replace: true })
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) setError('This invitation is no longer valid.')
      else if (err instanceof ApiError && (err.status === 403 || err.status === 409)) setError(err.message)
      else setError('Could not accept the invitation. Please try again.')
    }
  }

  const summary = (
    <>
      <h1 className="text-xl font-bold">You're invited</h1>
      <p className="mt-3">
        {who} invited you to join <strong>{invitation.workspace_name}</strong> as{' '}
        <strong>{invitation.role}</strong>.
      </p>
      <p className="mt-1 text-sm text-slate-400">
        This invitation is for <strong className="text-slate-200">{invitation.email}</strong>.
      </p>
    </>
  )

  if (state.status === 'anonymous') {
    // Remember this page, so logging in or registering brings you straight back.
    const back = { from: location }
    return (
      <Card>
        {summary}
        <p className="mt-4 text-sm text-slate-300">Log in or create an account with that email address to join.</p>
        <div className="mt-4 flex gap-3">
          <Link to="/login" state={back} className="rounded-md bg-indigo-500 px-4 py-2 font-medium hover:bg-indigo-400">
            Log in
          </Link>
          <Link to="/register" state={back} className="rounded-md bg-slate-700 px-4 py-2 hover:bg-slate-600">
            Create account
          </Link>
        </div>
      </Card>
    )
  }

  // Logged in from here on.
  const myEmail = state.status === 'authenticated' ? state.user.email : ''
  if (myEmail.toLowerCase() !== invitation.email.toLowerCase()) {
    return (
      <Card>
        {summary}
        <p role="alert" className="mt-4 rounded-md bg-amber-500/10 px-3 py-2 text-sm text-amber-300">
          You are logged in as <strong>{myEmail}</strong>, but this invitation was sent to{' '}
          <strong>{invitation.email}</strong>. Log out and use that account.
        </p>
        <button onClick={logout} className="mt-4 rounded-md bg-slate-700 px-4 py-2 hover:bg-slate-600">
          Log out
        </button>
      </Card>
    )
  }

  return (
    <Card>
      {summary}
      {error && (
        <p role="alert" className="mt-4 rounded-md bg-red-500/10 px-3 py-2 text-sm text-red-300">
          {error}
        </p>
      )}
      <button
        onClick={onAccept}
        disabled={accept.isPending}
        className="mt-5 w-full rounded-md bg-indigo-500 px-4 py-2 font-medium hover:bg-indigo-400 disabled:opacity-60"
      >
        {accept.isPending ? 'Joining…' : `Join ${invitation.workspace_name}`}
      </button>
    </Card>
  )
}

function Card({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-900 px-4 text-white">
      <div className="w-full max-w-md rounded-xl bg-slate-800 p-8 shadow-lg">{children}</div>
    </div>
  )
}