import { useState } from 'react'
import { Link } from 'react-router-dom'
import { NewProjectForm } from '../components/NewProjectForm'
import { QueryError } from '../components/QueryError'
import { WorkspaceGate } from '../components/WorkspaceGate'
import { WorkspaceHeader } from '../components/WorkspaceHeader'
import { can } from '../lib/permissions'
import { useProjects } from '../lib/projects'
import type { Workspace } from '../lib/workspaces'

export function ProjectsPage() {
  return <WorkspaceGate>{(workspace) => <Projects workspace={workspace} />}</WorkspaceGate>
}

function Projects({ workspace }: { workspace: Workspace }) {
  const { data: projects, isPending, isError, refetch } = useProjects(workspace.id)
  const [creating, setCreating] = useState(false)
  const canCreate = can(workspace.role, 'project:manage')

  return (
    <div>
      <WorkspaceHeader workspace={workspace} />

      <div className="mt-8 flex items-center justify-between">
        <h2 className="text-xl font-semibold">Projects</h2>
        {canCreate && !creating && (
          <button
            onClick={() => setCreating(true)}
            className="rounded-md bg-indigo-500 px-4 py-2 text-sm font-medium hover:bg-indigo-400"
          >
            New project
          </button>
        )}
      </div>

      {creating && (
        <div className="mt-4">
          <NewProjectForm workspaceId={workspace.id} workspaceSlug={workspace.slug} onCancel={() => setCreating(false)} />
        </div>
      )}

      {isPending && <p className="mt-4 text-slate-400">Loading…</p>}
      {isError && (
        <div className="mt-4">
          <QueryError message="Could not load the projects." onRetry={() => void refetch()} />
        </div>
      )}

      {projects && projects.length === 0 && !creating && (
        <p className="mt-6 text-slate-400">
          {canCreate
            ? 'No projects yet. Create the first one to start adding tasks.'
            : 'No projects yet. An admin or owner can create one.'}
        </p>
      )}

      {projects && projects.length > 0 && (
        <ul className="mt-4 grid gap-4 sm:grid-cols-2">
          {projects.map((project) => (
            <li key={project.id}>
              <Link
                to={`/w/${workspace.slug}/projects/${project.key}`}
                className="block rounded-xl bg-slate-800 p-5 ring-1 ring-slate-700 hover:ring-indigo-400"
              >
                <span className="rounded bg-indigo-500/20 px-2 py-0.5 text-xs font-medium text-indigo-300">
                  {project.key}
                </span>
                <p className="mt-3 font-semibold">{project.name}</p>
                {project.description && (
                  <p className="mt-1 line-clamp-2 text-sm text-slate-400">{project.description}</p>
                )}
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}