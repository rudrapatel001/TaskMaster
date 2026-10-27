import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './api'
import { applyMove } from './board'

export interface Task {
  id: string
  identifier: string // "WEB-12"
  number: number
  title: string
  description: string
  status_id: string // which board column it is in
  priority: 'low' | 'medium' | 'high' | 'urgent'
  assignee_id: string | null
  reporter_id: string | null
  due_date: string | null // "2026-10-20"
  position: number // order within its column
  created_at: string
  updated_at: string
}

const tasksKey = (workspaceId: string, projectId: string) =>
  ['workspaces', workspaceId, 'projects', projectId, 'tasks'] as const

/** The tasks of a project (up to 200, which is plenty for a board). */
export function useTasks(workspaceId: string, projectId: string) {
  return useQuery({
    queryKey: tasksKey(workspaceId, projectId),
    queryFn: () => api<Task[]>(`/workspaces/${workspaceId}/projects/${projectId}/tasks?limit=200`),
  })
}

export function useCreateTask(workspaceId: string, projectId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { title: string; status_id: string }) =>
      api<Task>(`/workspaces/${workspaceId}/projects/${projectId}/tasks`, { method: 'POST', body: input }),
    // The server puts a new task at the bottom of its column, so adding it to the end is right.
    onSuccess: (created) => {
      queryClient.setQueryData<Task[]>(tasksKey(workspaceId, projectId), (old) => [...(old ?? []), created])
    },
  })
}

/** The fields of a task that can be changed. null clears the two optional ones. */
export type TaskPatch = Partial<{
  title: string
  description: string
  priority: Task['priority']
  assignee_id: string | null
  due_date: string | null
}>

export function useUpdateTask(workspaceId: string, projectId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ number, patch }: { number: number; patch: TaskPatch }) =>
      api<Task>(`/workspaces/${workspaceId}/projects/${projectId}/tasks/${number}`, { method: 'PATCH', body: patch }),
    onSuccess: (updated) => {
      queryClient.setQueryData<Task[]>(tasksKey(workspaceId, projectId), (old) =>
        (old ?? []).map((task) => (task.id === updated.id ? updated : task)),
      )
    },
  })
}

export function useDeleteTask(workspaceId: string, projectId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (number: number) =>
      api<void>(`/workspaces/${workspaceId}/projects/${projectId}/tasks/${number}`, { method: 'DELETE' }),
    onSuccess: (_result, number) => {
      queryClient.setQueryData<Task[]>(tasksKey(workspaceId, projectId), (old) =>
        (old ?? []).filter((task) => task.number !== number),
      )
    },
  })
}

/**
 * Moving a card. The board updates instantly (an "optimistic" update) and the
 * request happens in the background. If the server says no, the board goes back
 * to how it was. Afterwards the tasks are reloaded, so the board always ends up
 * matching the server.
 */
export function useMoveTask(workspaceId: string, projectId: string) {
  const queryClient = useQueryClient()
  const key = tasksKey(workspaceId, projectId)

  const mutation = useMutation({
    mutationFn: (input: { number: number; statusId: string; before: number | null; snapshot: Task[] }) =>
      api<Task>(`/workspaces/${workspaceId}/projects/${projectId}/tasks/${input.number}/move`, {
        method: 'POST',
        body: { status_id: input.statusId, before: input.before },
      }),
    onSuccess: (moved) => {
      queryClient.setQueryData<Task[]>(key, (old) => (old ?? []).map((task) => (task.id === moved.id ? moved : task)))
    },
    onError: (_error, input) => {
      queryClient.setQueryData<Task[]>(key, input.snapshot) // put the board back
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: key }),
  })

  function move(input: { number: number; statusId: string; before: number | null }) {
    const snapshot = queryClient.getQueryData<Task[]>(key)
    if (!snapshot) return
    void queryClient.cancelQueries({ queryKey: key }) // a reload already in flight must not overwrite the new order
    queryClient.setQueryData<Task[]>(key, applyMove(snapshot, input.number, input.statusId, input.before))
    mutation.mutate({ ...input, snapshot })
  }

  return { move, isError: mutation.isError, reset: mutation.reset }
}