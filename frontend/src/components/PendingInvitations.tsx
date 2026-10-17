import { useState } from 'react'
import { QueryError } from './QueryError'
import { useInvitations, useRevokeInvitation } from '../lib/members'

/** The invitations that are still open, each with a Revoke button. */
export function PendingInvitations({ workspaceId }: { workspaceId: string }) {
  const { data: invitations, isPending, isError, refetch } = useInvitations(workspaceId)
  const revoke = useRevokeInvitation(workspaceId)
  const [error, setError] = useState<string | null>(null)

  async function onRevoke(id: string) {
    setError(null)
    try {
      await revoke.mutateAsync(id)
    } catch {
      setError('Could not revoke the invitation. It may already have been used.')
      void refetch()
    }
  }

  return (
    <div className="rounded-xl bg-slate-800 p-6">
      <h2 className="font-semibold">Pending invitations</h2>
      <p className="mt-1 text-sm text-slate-400">
        Links can't be shown again. To get a new link, invite the same email address again.
      </p>

      {error && (
        <p role="alert" className="mt-3 rounded-md bg-red-500/10 px-3 py-2 text-sm text-red-300">
          {error}
        </p>
      )}

      {isPending && <p className="mt-4 text-sm text-slate-400">Loading…</p>}
      {isError && (
        <div className="mt-4">
          <QueryError message="Could not load the invitations." onRetry={() => void refetch()} />
        </div>
      )}
      {invitations && invitations.length === 0 && (
        <p className="mt-4 text-sm text-slate-400">No pending invitations.</p>
      )}
      {invitations && invitations.length > 0 && (
        <ul className="mt-4 divide-y divide-slate-700">
          {invitations.map((invitation) => (
            <li key={invitation.id} className="flex items-center justify-between gap-4 py-3 text-sm">
              <div className="min-w-0">
                <p className="truncate">{invitation.email}</p>
                <p className="text-slate-400">
                  {invitation.role}
                  {invitation.invited_by && <> · invited by {invitation.invited_by}</>} · expires{' '}
                  {new Date(invitation.expires_at).toLocaleDateString()}
                </p>
              </div>
              <button
                onClick={() => onRevoke(invitation.id)}
                aria-label={`Revoke invitation for ${invitation.email}`}
                className="rounded-md bg-slate-700 px-3 py-1.5 hover:bg-slate-600"
              >
                Revoke
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}