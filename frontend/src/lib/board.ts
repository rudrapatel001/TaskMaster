import type { Status } from './projects'
import type { Task } from './tasks'

/** Task ids of each column, by column (status) id, in display order. */
export type Columns = Record<string, string[]>

/** Orders tasks inside a column. The id breaks ties, exactly like the server does. */
export function byPosition(a: Task, b: Task): number {
  return a.position - b.position || a.id.localeCompare(b.id)
}

/** Sorts tasks into their columns. */
export function columnsFromTasks(tasks: Task[], statuses: Status[]): Columns {
  const columns: Columns = {}
  for (const status of statuses) {
    columns[status.id] = tasks
      .filter((task) => task.status_id === status.id)
      .sort(byPosition)
      .map((task) => task.id)
  }
  return columns
}

/** The id of the column an item is in. The id may be a column's own id or a task's id. */
export function findColumn(columns: Columns, id: string): string | undefined {
  if (id in columns) return id
  return Object.keys(columns).find((columnId) => columns[columnId].includes(id))
}

/**
 * The task list as it will look after a move, worked out in the browser so the
 * card can jump straight to its new place while the server catches up.
 *
 * `before` is the number of the task the card goes directly above, or null for
 * the bottom of the column. This repeats the server's arithmetic: halfway
 * between the neighbours' positions.
 */
export function applyMove(tasks: Task[], number: number, statusId: string, before: number | null): Task[] {
  const moved = tasks.find((task) => task.number === number)
  if (!moved) return tasks

  const column = tasks.filter((task) => task.status_id === statusId && task.id !== moved.id).sort(byPosition)
  const beforeIndex = before === null ? -1 : column.findIndex((task) => task.number === before)

  let position: number
  if (beforeIndex === -1) {
    position = (column.length > 0 ? column[column.length - 1].position : 0) + 1000
  } else {
    const above = beforeIndex > 0 ? column[beforeIndex - 1].position : 0
    position = (above + column[beforeIndex].position) / 2
  }

  return tasks.map((task) => (task.id === moved.id ? { ...task, status_id: statusId, position } : task))
}