import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useCurrentWorkspace } from '../lib/workspaces'
import type { Workspace } from '../lib/workspaces'
import { QueryError } from './QueryError'

/**
 * Handles the loading, error and "not found" cases for the workspace in the
 * URL, so a page only has to describe what to show once it has a workspace.
 */
export function WorkspaceGate({ children }: { children: (workspace: Workspace) => ReactNode }) {
  const { workspace, isPending, isError, refetch } = useCurrentWorkspace()

  if (isPending) return <p className="text-slate-400">Loading…</p>
  if (isError) return <QueryError message="Could not load this workspace." onRetry={() => void refetch()} />

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

  return <>{children(workspace)}</>
}