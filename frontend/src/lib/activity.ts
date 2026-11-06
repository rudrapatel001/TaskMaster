import { useInfiniteQuery } from '@tanstack/react-query'
import { apiPage } from './api'

/** One entry of the feed: something somebody did. */
export interface ActivityEvent {
  id: string
  action: string // like "task.moved"
  entity_type: string
  entity_id: string
  actor_id: string | null
  actor_name: string
  metadata: Record<string, unknown> // a snapshot: identifier, title, ... as they were at the time
  created_at: string
}

interface ActivityPage {
  events: ActivityEvent[]
  nextCursor: string | null // null on the last page
}

const PAGE_SIZE = 30

async function fetchPage(path: string, cursor: string | null): Promise<ActivityPage> {
  const query = new URLSearchParams({ limit: String(PAGE_SIZE) })
  if (cursor) query.set('before', cursor)
  const { data, meta } = await apiPage<ActivityEvent[], { next_cursor: string | null }>(`${path}?${query}`)
  return { events: data, nextCursor: meta.next_cursor }
}

/** Everything that happened in a workspace, newest first, loaded a page at a time. */
export function useWorkspaceActivity(workspaceId: string) {
  return useInfiniteQuery({
    queryKey: ['workspaces', workspaceId, 'activity'] as const,
    queryFn: ({ pageParam }) => fetchPage(`/workspaces/${workspaceId}/activity`, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => lastPage.nextCursor,
  })
}

/** What happened to one task. */
export function useTaskActivity(workspaceId: string, projectId: string, taskNumber: number) {
  return useInfiniteQuery({
    queryKey: ['workspaces', workspaceId, 'projects', projectId, 'task-activity', taskNumber] as const,
    queryFn: ({ pageParam }) =>
      fetchPage(`/workspaces/${workspaceId}/projects/${projectId}/tasks/${taskNumber}/activity`, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => lastPage.nextCursor,
  })
}