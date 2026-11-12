import { useDroppable } from '@dnd-kit/core'
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable'
import type { Status } from '../lib/projects'
import type { Task } from '../lib/tasks'
import { AddTaskForm } from './AddTaskForm'
import { SortableTaskCard } from './SortableTaskCard'

/** One column of the board: a drop target holding sortable cards. */
export function BoardColumn({
  status,
  taskIds,
  tasksById,
  nameById,
  today,
  canWrite,
  isFirst,
  workspaceId,
  projectId,
  onOpenTask,
}: {
  status: Status
  taskIds: string[]
  tasksById: Map<string, Task>
  nameById: Map<string, string>
  today: string
  canWrite: boolean
  isFirst: boolean
  workspaceId: string
  projectId: string
  onOpenTask: (number: number) => void
}) {
  // Registering the whole column as a drop target lets an EMPTY column accept cards.
  const { setNodeRef, isOver } = useDroppable({ id: status.id })

  return (
    <section
      aria-label={status.name}
      className={`w-72 shrink-0 rounded-xl bg-slate-900 p-3 ring-1 ${isOver ? 'ring-indigo-400' : 'ring-slate-800'}`}
    >
      <h3 className="flex items-center justify-between px-1 text-sm font-semibold text-slate-300">
        {status.name}
        <span className="rounded-full bg-slate-800 px-2 py-0.5 text-xs font-normal text-slate-400">
          {taskIds.length}
        </span>
      </h3>

      <div ref={setNodeRef} className="mt-3 min-h-10">
        <SortableContext items={taskIds} strategy={verticalListSortingStrategy}>
          <ul className="space-y-2">
            {taskIds.map((id) => {
              const task = tasksById.get(id)
              if (!task) return null
              return (
                <SortableTaskCard
                  key={id}
                  task={task}
                  assigneeName={task.assignee_id ? nameById.get(task.assignee_id) : undefined}
                  done={status.category === 'done'}
                  today={today}
                  draggable={canWrite}
                  onOpen={() => onOpenTask(task.number)}
                />
              )
            })}
          </ul>
        </SortableContext>
      </div>

      {canWrite && (
        <AddTaskForm
          workspaceId={workspaceId}
          projectId={projectId}
          statusId={status.id}
          listenForShortcut={isFirst}
        />
      )}
    </section>
  )
}