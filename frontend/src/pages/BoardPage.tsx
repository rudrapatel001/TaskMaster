import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { AddTaskForm } from '../components/AddTaskForm'
import { QueryError } from '../components/QueryError'
import { TaskCard } from '../components/TaskCard'
import { WorkspaceGate } from '../components/WorkspaceGate'
import { WorkspaceHeader } from '../components/WorkspaceHeader'
import { useMembers } from '../lib/members'
import { can } from '../lib/permissions'
import { useProject, useProjects } from '../lib/projects'
import type { Project } from '../lib/projects'
import { useTasks } from '../lib/tasks'
import type { Workspace } from '../lib/workspaces'

export function BoardPage() {
  const { key = '' } = useParams()
  return <WorkspaceGate>{(workspace) => <FindProject workspace={workspace} projectKey={key} />}</WorkspaceGate>
}

/** Finds the project whose key is in the URL, then shows its board. */
function FindProject({ workspace, projectKey }: { workspace: Workspace; projectKey: string }) {
  const { data: projects, isPending, isError, refetch } = useProjects(workspace.id)

  if (isPending) return <p className="text-slate-400">Loading…</p>
  if (isError) return <QueryError message="Could not load the projects." onRetry={() => void refetch()} />

  const project = projects.find((p) => p.key === projectKey.toUpperCase())
  if (!project) {
    return (
      <div>
        <WorkspaceHeader workspace={workspace} />
        <h2 className="mt-8 text-xl font-semibold">Project not found</h2>
        <Link to={`/w/${workspace.slug}/projects`} className="mt-3 inline-block text-indigo-300 hover:underline">
          Back to projects
        </Link>
      </div>
    )
  }

  return <Board workspace={workspace} project={project} />
}

function Board({ workspace, project }: { workspace: Workspace; project: Project }) {
  const detail = useProject(workspace.id, project.id)
  const tasks = useTasks(workspace.id, project.id)
  const members = useMembers(workspace.id)
  const canWrite = can(workspace.role, 'task:write')
  const [today] = useState(() => new Date().toLocaleDateString('en-CA')) // decided once, so every card agrees

  // Turn the member list into "user id -> name", to show who a task is assigned to.
  const nameById = new Map((members.data ?? []).map((member) => [member.user_id, member.name]))

  return (
    <div>
      <WorkspaceHeader workspace={workspace} />

      <div className="mt-6">
        <Link to={`/w/${workspace.slug}/projects`} className="text-sm text-slate-400 hover:text-white">
          ← Projects
        </Link>
        <div className="mt-2 flex items-center gap-3">
          <span className="rounded bg-indigo-500/20 px-2 py-0.5 text-sm font-medium text-indigo-300">{project.key}</span>
          <h2 className="text-2xl font-semibold">{project.name}</h2>
        </div>
        {project.description && <p className="mt-1 text-sm text-slate-400">{project.description}</p>}
      </div>

      {(detail.isPending || tasks.isPending) && <p className="mt-6 text-slate-400">Loading the board…</p>}
      {(detail.isError || tasks.isError) && (
        <div className="mt-6">
          <QueryError
            message="Could not load the board."
            onRetry={() => {
              void detail.refetch()
              void tasks.refetch()
            }}
          />
        </div>
      )}

      {detail.data && tasks.data && (
        <div className="mt-6 flex items-start gap-4 overflow-x-auto pb-4">
          {detail.data.statuses.map((status) => {
            // Cards in this column, in their saved order.
            const cards = tasks.data
              .filter((task) => task.status_id === status.id)
              .sort((a, b) => a.position - b.position)

            return (
              <section
                key={status.id}
                aria-label={status.name}
                className="w-72 shrink-0 rounded-xl bg-slate-900 p-3 ring-1 ring-slate-800"
              >
                <h3 className="flex items-center justify-between px-1 text-sm font-semibold text-slate-300">
                  {status.name}
                  <span className="rounded-full bg-slate-800 px-2 py-0.5 text-xs font-normal text-slate-400">
                    {cards.length}
                  </span>
                </h3>

                <ul className="mt-3 space-y-2">
                  {cards.map((task) => (
                    <TaskCard
                      key={task.id}
                      task={task}
                      assigneeName={task.assignee_id ? nameById.get(task.assignee_id) : undefined}
                      done={status.category === 'done'}
                      today={today}
                    />
                  ))}
                </ul>

                {canWrite && <AddTaskForm workspaceId={workspace.id} projectId={project.id} statusId={status.id} />}
              </section>
            )
          })}
        </div>
      )}
    </div>
  )
}