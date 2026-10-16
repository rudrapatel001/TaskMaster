import { Link, useParams } from 'react-router-dom'
import { QueryError } from '../components/QueryError'
import { RoleBadge } from '../components/RoleBadge'
import { WorkspaceSettings } from '../components/WorkspaceSettings'
import { useWorkspaces } from '../lib/workspaces'

export function WorkspacePage() {
  const { slug } = useParams()
  const { data: workspaces, isPending, isError, refetch } = useWorkspaces()

  if (isPending) return <p className="text-slate-400">Loading…</p>
  if (isError) {
    return <QueryError message="Could not load this workspace." onRetry={() => void refetch()} />
  }

  const workspace = workspaces.find((w) => w.slug === slug)
  if (!workspace) {
    return (
      <div>
        <h1 className="text-2xl font-bold">Workspace not found</h1>
        <p className="mt-2 text-slate-400">It may have been deleted, or you may not be a member.</p>
        <Link to="/" className="mt-4 inline-block text-indigo-300 hover:underline">
          Go to your workspaces
        </Link>
      </div>
    )
  }

  return (
    <div>
      <div className="flex items-center gap-3">
        <h1 className="text-3xl font-bold">{workspace.name}</h1>
        <RoleBadge role={workspace.role} />
      </div>
      <p className="mt-2 text-slate-400">Projects and tasks will appear here.</p>

      {/* key: switching workspaces resets the forms below instead of reusing their old values */}
      <WorkspaceSettings key={workspace.id} workspace={workspace} />
    </div>
  )
}