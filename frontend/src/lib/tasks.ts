import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './api'

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