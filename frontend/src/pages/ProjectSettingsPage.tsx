import { Link, useParams } from 'react-router-dom'
import { ColumnsSettings } from '../components/ColumnsSettings'
import { DeleteProjectSection } from '../components/DeleteProjectSection'
import { ProjectSettingsForm } from '../components/ProjectSettingsForm'
import { QueryError } from '../components/QueryError'
import { WorkspaceGate } from '../components/WorkspaceGate'
import { WorkspaceHeader } from '../components/WorkspaceHeader'
import { can } from '../lib/permissions'
import { useProjects } from '../lib/projects'
import type { Workspace } from '../lib/workspaces'

export function ProjectSettingsPage() {
  const { key = '' } = useParams()
  return <WorkspaceGate>{(workspace) => <Settings workspace={workspace} projectKey={key} />}</WorkspaceGate>
}

function Settings({ workspace, projectKey }: { workspace: Workspace; projectKey: string }) {
  const { data: projects, isPending, isError, refetch } = useProjects(workspace.id)
  const projectsUrl = `/w/${workspace.slug}/projects`

  if (isPending) return <p className="text-slate-400">Loading…</p>
  if (isError) return <QueryError message="Could not load the projects." onRetry={() => void refetch()} />

  const project = projects.find((p) => p.key === projectKey.toUpperCase())
  if (!project) {
    return (
      <div>
        <WorkspaceHeader workspace={workspace} />
        <h2 className="mt-8 text-xl font-semibold">Project not found</h2>
        <Link to={projectsUrl} className="mt-3 inline-block text-indigo-300 hover:underline">
          Back to projects
        </Link>
      </div>
    )
  }

  // The buttons are hidden from people who may not use them, and the server refuses them anyway.
  if (!can(workspace.role, 'project:manage')) {
    return (
      <div>
        <WorkspaceHeader workspace={workspace} />
        <h2 className="mt-8 text-xl font-semibold">Project settings</h2>
        <p className="mt-2 text-slate-400">Only admins and owners can change a project's settings.</p>
        <Link to={`${projectsUrl}/${project.key}`} className="mt-3 inline-block text-indigo-300 hover:underline">
          Back to the board
        </Link>
      </div>
    )
  }

  return (
    <div>
      <WorkspaceHeader workspace={workspace} />

      <div className="mt-6">
        <Link to={`${projectsUrl}/${project.key}`} className="text-sm text-slate-400 hover:text-white">
          ← Back to {project.name}
        </Link>
        <h2 className="mt-2 text-2xl font-semibold">Project settings</h2>
      </div>

      <div className="mt-6 max-w-2xl space-y-6">
        {/* key: switching project starts the form fresh instead of keeping the old values */}
        <ProjectSettingsForm key={project.id} workspaceId={workspace.id} project={project} />
        <ColumnsSettings key={`columns-${project.id}`} workspaceId={workspace.id} projectId={project.id} />
        <DeleteProjectSection workspaceId={workspace.id} workspaceSlug={workspace.slug} project={project} />
      </div>
    </div>
  )
}