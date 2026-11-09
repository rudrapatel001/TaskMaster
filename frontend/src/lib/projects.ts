import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './api'

export interface Project {
  id: string
  key: string // short and permanent, like "WEB": it prefixes every task id (WEB-12)
  name: string
  description: string
  created_at: string
  updated_at: string
}

/** One column of a project's board. */
export interface Status {
  id: string
  name: string
  category: 'todo' | 'in_progress' | 'done' // what the column means, whatever it is called
  position: number
}

export interface ProjectDetail extends Project {
  statuses: Status[]
}

const projectsKey = (workspaceId: string) => ['workspaces', workspaceId, 'projects'] as const

/** The projects of a workspace. */
export function useProjects(workspaceId: string) {
  return useQuery({
    queryKey: projectsKey(workspaceId),
    queryFn: () => api<Project[]>(`/workspaces/${workspaceId}/projects`),
  })
}

/** One project with its board columns. */
export function useProject(workspaceId: string, projectId: string) {
  return useQuery({
    queryKey: [...projectsKey(workspaceId), projectId] as const,
    queryFn: () => api<ProjectDetail>(`/workspaces/${workspaceId}/projects/${projectId}`),
  })
}

export function useCreateProject(workspaceId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { key: string; name: string; description: string }) =>
      api<ProjectDetail>(`/workspaces/${workspaceId}/projects`, { method: 'POST', body: input }),
    // Reload only the list; the new project's details are fetched when it is opened.
    onSuccess: () => queryClient.invalidateQueries({ queryKey: projectsKey(workspaceId), exact: true }),
  })
}

/** Changes a project's name and description. Its key can never change. */
export function useUpdateProject(workspaceId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, name, description }: { id: string; name: string; description: string }) =>
      api<Project>(`/workspaces/${workspaceId}/projects/${id}`, { method: 'PATCH', body: { name, description } }),
    // Update the list and, if it is loaded, the project's own page, so the new name shows everywhere at once.
    onSuccess: (updated) => {
      queryClient.setQueryData<Project[]>(projectsKey(workspaceId), (old) =>
        (old ?? []).map((project) => (project.id === updated.id ? updated : project)),
      )
      queryClient.setQueryData<ProjectDetail>([...projectsKey(workspaceId), updated.id], (old) =>
        old ? { ...old, ...updated } : old,
      )
    },
  })
}

/** Deletes a project, with all its tasks, comments and columns. */
export function useDeleteProject(workspaceId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api<void>(`/workspaces/${workspaceId}/projects/${id}`, { method: 'DELETE' }),
    onSuccess: (_result, id) => {
      queryClient.setQueryData<Project[]>(projectsKey(workspaceId), (old) =>
        (old ?? []).filter((project) => project.id !== id),
      )
      // Forget everything cached under this project: its columns, tasks, comments and feed.
      queryClient.removeQueries({ queryKey: [...projectsKey(workspaceId), id] })
    },
  })
}

/** Suggests a project key from its name: "Website" becomes "WEB". Empty if no sensible key. */
export function suggestKey(name: string): string {
  const key = name.toUpperCase().replace(/[^A-Z0-9]/g, '').slice(0, 3)
  return /^[A-Z]/.test(key) && key.length >= 2 ? key : ''
}