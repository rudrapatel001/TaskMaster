import type { Workspace } from '../lib/workspaces'
import { RoleBadge } from './RoleBadge'
import { WorkspaceNav } from './WorkspaceNav'

/** The workspace's name, your role in it, and its tabs. */
export function WorkspaceHeader({ workspace }: { workspace: Workspace }) {
  return (
    <>
      <div className="flex items-center gap-3">
        <h1 className="text-3xl font-bold">{workspace.name}</h1>
        <RoleBadge role={workspace.role} />
      </div>
      <WorkspaceNav slug={workspace.slug} />
    </>
  )
}