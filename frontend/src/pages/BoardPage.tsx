import { useCallback, useMemo, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import {
  DndContext,
  DragOverlay,
  KeyboardSensor,
  PointerSensor,
  closestCorners,
  useSensor,
  useSensors,
} from '@dnd-kit/core'
import type { DragEndEvent, DragOverEvent, DragStartEvent } from '@dnd-kit/core'
import { arrayMove, sortableKeyboardCoordinates } from '@dnd-kit/sortable'
import { useAuth } from '../auth/AuthContext'
import { BoardColumn } from '../components/BoardColumn'
import { QueryError } from '../components/QueryError'
import { TaskCard } from '../components/TaskCard'
import { TaskPanel } from '../components/TaskPanel'
import { WorkspaceGate } from '../components/WorkspaceGate'
import { WorkspaceHeader } from '../components/WorkspaceHeader'
import { columnsFromTasks, findColumn } from '../lib/board'
import type { Columns } from '../lib/board'
import { useMembers } from '../lib/members'
import type { Member } from '../lib/members'
import { can } from '../lib/permissions'
import { useProject, useProjects } from '../lib/projects'
import type { Project, Status } from '../lib/projects'
import { useMoveTask, useTasks } from '../lib/tasks'
import type { Task } from '../lib/tasks'
import type { Workspace } from '../lib/workspaces'

// Stable empty values, so hooks that depend on them do not re-run on every render.
const NO_TASKS: Task[] = []
const NO_STATUSES: Status[] = []
const NO_MEMBERS: Member[] = []

function sameOrder(a: string[] | undefined, b: string[]): boolean {
  return a !== undefined && a.length === b.length && a.every((id, index) => id === b[index])
}

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
  const { move, isError: moveFailed, reset: resetMoveError } = useMoveTask(workspace.id, project.id)
  const { state: auth } = useAuth()
  const canWrite = can(workspace.role, 'task:write')
  const [today] = useState(() => new Date().toLocaleDateString('en-CA')) // decided once, so every card agrees

  const taskList = tasks.data ?? NO_TASKS
  const statuses = detail.data?.statuses ?? NO_STATUSES
  const memberList = members.data ?? NO_MEMBERS

  const tasksById = useMemo(() => new Map(taskList.map((task) => [task.id, task])), [taskList])
  const nameById = useMemo(() => new Map(memberList.map((member) => [member.user_id, member.name])), [memberList])

  // The columns as the server has them. While a card is being dragged, a
  // working copy (dragColumns) shows where it would land.
  const baseColumns = useMemo(() => columnsFromTasks(taskList, statuses), [taskList, statuses])
  const [dragColumns, setDragColumns] = useState<Columns | null>(null)
  const [activeId, setActiveId] = useState<string | null>(null)
  const columns = dragColumns ?? baseColumns

  // The open task lives in the URL (?task=12), so it can be linked to and Back closes it.
  const [params, setParams] = useSearchParams()
  const openNumber = Number(params.get('task'))
  const openTask = Number.isInteger(openNumber) && openNumber > 0 ? taskList.find((t) => t.number === openNumber) : undefined
  const openPanel = (number: number) => setParams({ task: String(number) })
  const closePanel = useCallback(() => setParams({}), [setParams])

  const sensors = useSensors(
    // A drag starts after moving 5 pixels, so a plain click still opens the card.
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    // Space picks a card up and drops it; Enter stays free for opening it.
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
      keyboardCodes: { start: ['Space'], cancel: ['Escape'], end: ['Space'] },
    }),
  )

  function onDragStart(event: DragStartEvent) {
    setActiveId(String(event.active.id))
    setDragColumns(baseColumns)
  }

  // While dragging over another column, show the card there straight away.
  function onDragOver({ active, over }: DragOverEvent) {
    if (!over) return
    const dragged = String(active.id)
    const overId = String(over.id)

    setDragColumns((current) => {
      const cols = current ?? baseColumns
      const from = findColumn(cols, dragged)
      const to = findColumn(cols, overId)
      if (!from || !to || from === to) return current

      const target = cols[to]
      let index: number
      if (overId in cols) {
        index = target.length // over the column itself: the bottom
      } else {
        const rect = active.rect.current.translated
        const below = rect !== null && rect.top > over.rect.top + over.rect.height
        const overIndex = target.indexOf(overId)
        index = overIndex >= 0 ? overIndex + (below ? 1 : 0) : target.length
      }

      return {
        ...cols,
        [from]: cols[from].filter((id) => id !== dragged),
        [to]: [...target.slice(0, index), dragged, ...target.slice(index)],
      }
    })
  }

  function onDragEnd({ active, over }: DragEndEvent) {
    const cols = dragColumns ?? baseColumns
    setActiveId(null)
    setDragColumns(null)
    if (!over || !canWrite) return

    const dragged = String(active.id)
    const overId = String(over.id)
    const from = findColumn(cols, dragged)
    const to = findColumn(cols, overId)
    const task = tasksById.get(dragged)
    if (!from || !to || !task) return

    // Dropped within one column: apply the reorder.
    let ids = cols[to]
    if (from === to) {
      const oldIndex = ids.indexOf(dragged)
      const newIndex = overId in cols ? ids.length - 1 : ids.indexOf(overId)
      if (oldIndex !== -1 && newIndex !== -1 && oldIndex !== newIndex) ids = arrayMove(ids, oldIndex, newIndex)
    }

    // Dropped back where it started: nothing to do.
    if (findColumn(baseColumns, dragged) === to && sameOrder(baseColumns[to], ids)) return

    // "Put it directly above the card that now follows it" (or at the bottom).
    const nextId = ids[ids.indexOf(dragged) + 1]
    const before = nextId ? (tasksById.get(nextId)?.number ?? null) : null
    move({ number: task.number, statusId: to, before })
  }

  function onDragCancel() {
    setActiveId(null)
    setDragColumns(null)
  }

  const activeTask = activeId ? tasksById.get(activeId) : undefined
  const activeStatus = activeId ? statuses.find((s) => s.id === findColumn(columns, activeId)) : undefined

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

      {moveFailed && (
        <p role="alert" className="mt-4 flex items-center gap-3 rounded-md bg-red-500/10 px-3 py-2 text-sm text-red-300">
          The task could not be moved, so the board was put back.
          <button onClick={resetMoveError} className="underline">
            Dismiss
          </button>
        </p>
      )}

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
        <DndContext
          sensors={sensors}
          collisionDetection={closestCorners}
          onDragStart={onDragStart}
          onDragOver={onDragOver}
          onDragEnd={onDragEnd}
          onDragCancel={onDragCancel}
        >
          <div className="mt-6 flex items-start gap-4 overflow-x-auto pb-4">
            {statuses.map((status) => (
              <BoardColumn
                key={status.id}
                status={status}
                taskIds={columns[status.id] ?? []}
                tasksById={tasksById}
                nameById={nameById}
                today={today}
                canWrite={canWrite}
                workspaceId={workspace.id}
                projectId={project.id}
                onOpenTask={openPanel}
              />
            ))}
          </div>

          <DragOverlay>
            {activeTask ? (
              <div className="cursor-grabbing">
                <TaskCard
                  task={activeTask}
                  assigneeName={activeTask.assignee_id ? nameById.get(activeTask.assignee_id) : undefined}
                  done={activeStatus?.category === 'done'}
                  today={today}
                />
              </div>
            ) : null}
          </DragOverlay>
        </DndContext>
      )}

      {openTask && (
        <TaskPanel
          workspaceId={workspace.id}
          projectId={project.id}
          task={openTask}
          statuses={statuses}
          members={memberList}
          canWrite={canWrite}
          role={workspace.role}
          myUserId={auth.status === 'authenticated' ? auth.user.id : ''}
          onMove={(statusId) => move({ number: openTask.number, statusId, before: null })}
          onClose={closePanel}
        />
      )}
    </div>
  )
}