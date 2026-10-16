import { Navigate } from 'react-router-dom'
import { QueryError } from '../components/QueryError'
import { useWorkspaces } from '../lib/workspaces'

/** The page at "/": sends you to your first workspace, or to create one. */
export function RootRedirect() {
  const { data: workspaces, isPending, isError, refetch } = useWorkspaces()

  if (isPending) return <p className="text-slate-400">Loading…</p>
  if (isError) {
    return <QueryError message="Could not load your workspaces." onRetry={() => void refetch()} />
  }

  if (workspaces.length === 0) return <Navigate to="/workspaces/new" replace />
  return <Navigate to={`/w/${workspaces[0].slug}`} replace />
}