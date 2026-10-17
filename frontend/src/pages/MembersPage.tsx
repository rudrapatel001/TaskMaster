import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { InviteForm } from '../components/InviteForm'
import { PendingInvitations } from '../components/PendingInvitations'
import { QueryError } from '../components/QueryError'
import { RoleBadge } from '../components/RoleBadge'
import { WorkspaceNav } from '../components/WorkspaceNav'
import { ApiError } from '../lib/api'
import { useChangeRole, useMembers, useRemoveMember } from '../lib/members'
import type { Member } from '../lib/members'
import { can, canRemoveMember, rolesYouCanGrant, rolesYouCanInvite } from '../lib/permissions'
import type { Role } from '../lib/permissions'
import { useWorkspaces } from '../lib/workspaces'
import type { Workspace } from '../lib/workspaces'

export function MembersPage() {
  const { slug = '' } = useParams()
  const { state } = useAuth()
  const { data: workspaces, isPending, isError, refetch } = useWorkspaces()

  if (isPending) return <p className="text-slate-400">Loading…</p>
  if (isError) return <QueryError message="Could not load this workspace." onRetry={() => void refetch()} />

  const workspace = workspaces.find((w) => w.slug === slug)
  if (!workspace || state.status !== 'authenticated') {
    return (
      <div>
        <h1 className="text-2xl font-bold">Workspace not found</h1>
        <Link to="/" className="mt-4 inline-block text-indigo-300 hover:underline">
          Go to your workspaces
        </Link>
      </div>
    )
  }

  return <MembersContent key={workspace.id} workspace={workspace} myId={state.user.id} />
}

function MembersContent({ workspace, myId }: { workspace: Workspace; myId: string }) {
  const { data: members, isPending, isError, refetch } = useMembers(workspace.id)
  const changeRole = useChangeRole(workspace.id)
  const removeMember = useRemoveMember(workspace.id)
  const [error, setError] = useState<string | null>(null)

  async function run(action: () => Promise<unknown>) {
    setError(null)
    try {
      await action()
    } catch (err) {
      // Show the server's reason when it gave one (for example: the last owner cannot be demoted).
      if (err instanceof ApiError && [403, 409].includes(err.status)) setError(err.message)
      else setError('That did not work. The list has been refreshed.')
      void refetch() // the page may have been out of date
    }
  }

  const inviteRoles = rolesYouCanInvite(workspace.role)

  return (
    <div>
      <div className="flex items-center gap-3">
        <h1 className="text-3xl font-bold">{workspace.name}</h1>
        <RoleBadge role={workspace.role} />
      </div>
      <WorkspaceNav slug={workspace.slug} />

      <h2 className="mt-8 text-xl font-semibold">Members</h2>

      {error && (
        <p role="alert" className="mt-3 rounded-md bg-red-500/10 px-3 py-2 text-sm text-red-300">
          {error}
        </p>
      )}

      {isPending && <p className="mt-4 text-slate-400">Loading…</p>}
      {isError && (
        <div className="mt-4">
          <QueryError message="Could not load the members." onRetry={() => void refetch()} />
        </div>
      )}
      {members && (
        <ul className="mt-4 divide-y divide-slate-700 rounded-xl bg-slate-800">
          {members.map((member) => (
            <MemberRow
              key={member.user_id}
              member={member}
              myRole={workspace.role}
              isMe={member.user_id === myId}
              onChangeRole={(role) => run(() => changeRole.mutateAsync({ userId: member.user_id, role }))}
              onRemove={() => run(() => removeMember.mutateAsync(member.user_id))}
            />
          ))}
        </ul>
      )}

      {can(workspace.role, 'members:invite') && inviteRoles.length > 0 && (
        <div className="mt-10 space-y-6">
          <InviteForm workspaceId={workspace.id} roles={inviteRoles} />
          <PendingInvitations workspaceId={workspace.id} />
        </div>
      )}
    </div>
  )
}

function MemberRow({
  member,
  myRole,
  isMe,
  onChangeRole,
  onRemove,
}: {
  member: Member
  myRole: Role
  isMe: boolean
  onChangeRole: (role: Role) => void
  onRemove: () => void
}) {
  const [confirming, setConfirming] = useState(false)

  // You manage other people here. Leaving, or stepping down, is done from the Overview tab.
  const roleOptions = isMe ? [] : rolesYouCanGrant(myRole, member.role)
  const removable = !isMe && canRemoveMember(myRole, member.role)

  return (
    <li className="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
      <div className="min-w-0">
        <p className="truncate font-medium">
          {member.name}
          {isMe && <span className="ml-2 text-sm font-normal text-slate-400">(you)</span>}
        </p>
        <p className="truncate text-sm text-slate-400">{member.email}</p>
      </div>

      <div className="flex items-center gap-3">
        {roleOptions.length > 0 ? (
          <select
            aria-label={`Role for ${member.name}`}
            value={member.role}
            onChange={(event) => onChangeRole(event.target.value as Role)}
            className="rounded-md bg-slate-900 px-2 py-1 text-sm ring-1 ring-slate-600 focus:ring-2 focus:ring-indigo-400"
          >
            {roleOptions.map((role) => (
              <option key={role} value={role}>
                {role}
              </option>
            ))}
          </select>
        ) : (
          <RoleBadge role={member.role} />
        )}

        {removable &&
          (confirming ? (
            <span className="flex items-center gap-2 text-sm">
              <button
                onClick={() => {
                  setConfirming(false)
                  onRemove()
                }}
                className="rounded-md bg-red-500 px-2 py-1 font-medium hover:bg-red-400"
              >
                Remove {member.name}
              </button>
              <button onClick={() => setConfirming(false)} className="text-slate-300 hover:text-white">
                Cancel
              </button>
            </span>
          ) : (
            <button
              onClick={() => setConfirming(true)}
              aria-label={`Remove ${member.name}`}
              className="rounded-md bg-slate-700 px-2 py-1 text-sm hover:bg-slate-600"
            >
              Remove
            </button>
          ))}
      </div>
    </li>
  )
}