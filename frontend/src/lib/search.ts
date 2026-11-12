import { useQuery } from '@tanstack/react-query'
import { api } from './api'

export interface TaskSearchResult {
  identifier: string // "WEB-12"
  number: number
  title: string
  project_key: string
  project_name: string
}

/** Finds tasks by title, or by identifier such as "web-12". Only searches for 2 or more characters. */
export function useTaskSearch(workspaceId: string | undefined, query: string) {
  const text = query.trim()
  return useQuery({
    queryKey: ['workspaces', workspaceId, 'search', text] as const,
    queryFn: async () => {
      const params = new URLSearchParams({ q: text })
      const result = await api<{ tasks: TaskSearchResult[] }>(`/workspaces/${workspaceId}/search?${params}`)
      return result.tasks
    },
    enabled: workspaceId !== undefined && text.length >= 2,
    staleTime: 30_000, // typing the same thing again within 30 seconds reuses the answer
  })
}