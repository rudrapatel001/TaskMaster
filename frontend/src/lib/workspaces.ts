import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { api } from './api'
import type { Role } from './permissions'

export interface Workspace {
  id: string
  name: string
  slug: string
  role: Role // the logged-in user's role in this workspace
  created_at: string
  updated_at: string
}

// All workspace data lives in the query cache under this key.
export const workspacesKey = ['workspaces'] as const

/** The workspaces the logged-in user belongs to. */
export function useWorkspaces() {
  return useQuery({
    queryKey: workspacesKey,
    queryFn: () => api<Workspace[]>('/workspaces'),
  })
}

// After a change succeeds, the cache is updated by hand, so every part of the
// page that shows workspaces updates at once, without waiting for a refetch.

export function useCreateWorkspace() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => api<Workspace>('/workspaces', { method: 'POST', body: { name } }),
    onSuccess: (created) => {
      queryClient.setQueryData<Workspace[]>(workspacesKey, (old) => [...(old ?? []), created])
    },
  })
}

export function useRenameWorkspace() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      api<Workspace>(`/workspaces/${id}`, { method: 'PATCH', body: { name } }),
    onSuccess: (updated) => {
      queryClient.setQueryData<Workspace[]>(workspacesKey, (old) =>
        (old ?? []).map((w) => (w.id === updated.id ? updated : w)),
      )
    },
  })
}

function removeFromCache(queryClient: ReturnType<typeof useQueryClient>, id: string) {
  queryClient.setQueryData<Workspace[]>(workspacesKey, (old) => (old ?? []).filter((w) => w.id !== id))
}

export function useDeleteWorkspace() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api<void>(`/workspaces/${id}`, { method: 'DELETE' }),
    onSuccess: (_result, id) => removeFromCache(queryClient, id),
  })
}

export function useLeaveWorkspace() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api<void>(`/workspaces/${id}/leave`, { method: 'POST' }),
    onSuccess: (_result, id) => removeFromCache(queryClient, id),
  })
}

/** The workspace named in the URL (/w/:slug/...), found in the user's list. */
export function useCurrentWorkspace() {
  const { slug } = useParams()
  const { data, isPending, isError, refetch } = useWorkspaces()
  return { workspace: data?.find((w) => w.slug === slug), isPending, isError, refetch }
}