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

/** Suggests a project key from its name: "Website" becomes "WEB". Empty if no sensible key. */
export function suggestKey(name: string): string {
  const key = name.toUpperCase().replace(/[^A-Z0-9]/g, '').slice(0, 3)
  return /^[A-Z]/.test(key) && key.length >= 2 ? key : ''
}