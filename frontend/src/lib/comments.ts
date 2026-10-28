import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './api'

export interface Comment {
  id: string
  body: string
  author_id: string | null // null once the author's account is gone
  author_name: string
  created_at: string
  updated_at: string
}

// A separate branch from the task list's key, so reloading tasks does not reload comments.
const commentsKey = (workspaceId: string, projectId: string, taskNumber: number) =>
  ['workspaces', workspaceId, 'projects', projectId, 'task-comments', taskNumber] as const

const base = (workspaceId: string, projectId: string, taskNumber: number) =>
  `/workspaces/${workspaceId}/projects/${projectId}/tasks/${taskNumber}/comments`

/** The comments on a task, oldest first. */
export function useComments(workspaceId: string, projectId: string, taskNumber: number) {
  return useQuery({
    queryKey: commentsKey(workspaceId, projectId, taskNumber),
    queryFn: () => api<Comment[]>(base(workspaceId, projectId, taskNumber)),
  })
}

export function useCreateComment(workspaceId: string, projectId: string, taskNumber: number) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: string) =>
      api<Comment>(base(workspaceId, projectId, taskNumber), { method: 'POST', body: { body } }),
    onSuccess: (created) => {
      queryClient.setQueryData<Comment[]>(commentsKey(workspaceId, projectId, taskNumber), (old) => [
        ...(old ?? []),
        created,
      ])
    },
  })
}

export function useUpdateComment(workspaceId: string, projectId: string, taskNumber: number) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: string }) =>
      api<Comment>(`${base(workspaceId, projectId, taskNumber)}/${id}`, { method: 'PATCH', body: { body } }),
    onSuccess: (updated) => {
      queryClient.setQueryData<Comment[]>(commentsKey(workspaceId, projectId, taskNumber), (old) =>
        (old ?? []).map((comment) => (comment.id === updated.id ? updated : comment)),
      )
    },
  })
}

export function useDeleteComment(workspaceId: string, projectId: string, taskNumber: number) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      api<void>(`${base(workspaceId, projectId, taskNumber)}/${id}`, { method: 'DELETE' }),
    onSuccess: (_result, id) => {
      queryClient.setQueryData<Comment[]>(commentsKey(workspaceId, projectId, taskNumber), (old) =>
        (old ?? []).filter((comment) => comment.id !== id),
      )
    },
  })
}