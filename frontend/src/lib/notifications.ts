import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, apiPage } from './api'

/** One notification. (Called AppNotification because browsers already have a built-in "Notification".) */
export interface AppNotification {
  id: string
  type: string // "task.assigned" or "task.commented"
  actor_id: string | null
  actor_name: string
  metadata: Record<string, unknown> // a snapshot: identifier, title, workspace_slug, ...
  read_at: string | null // null while unread
  created_at: string
}

const countKey = ['notifications', 'unread-count'] as const
const listKey = ['notifications', 'list'] as const

const CHECK_EVERY_MS = 30_000

/**
 * How many notifications are unread: the number on the bell. It is checked
 * every 30 seconds while the tab is visible, and again when you come back to it.
 */
export function useUnreadCount() {
  return useQuery({
    queryKey: countKey,
    queryFn: async () => (await api<{ count: number }>('/notifications/unread-count')).count,
    refetchInterval: CHECK_EVERY_MS,
  })
}

interface NotificationPage {
  notifications: AppNotification[]
  nextCursor: string | null // null on the last page
}

/** The notifications themselves, newest first. Only loaded while the bell is open. */
export function useNotifications(enabled: boolean) {
  return useInfiniteQuery({
    queryKey: listKey,
    enabled,
    queryFn: async ({ pageParam }): Promise<NotificationPage> => {
      const query = new URLSearchParams({ limit: '20' })
      if (pageParam) query.set('before', pageParam)
      const { data, meta } = await apiPage<AppNotification[], { next_cursor: string | null }>(
        `/notifications?${query}`,
      )
      return { notifications: data, nextCursor: meta.next_cursor }
    },
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => lastPage.nextCursor,
  })
}

export function useMarkRead() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api<void>(`/notifications/${id}/read`, { method: 'POST' }),
    // Reload both the number on the bell and the list.
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: countKey }),
        queryClient.invalidateQueries({ queryKey: listKey }),
      ]),
  })
}

export function useMarkAllRead() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api<void>('/notifications/read-all', { method: 'POST' }),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: countKey }),
        queryClient.invalidateQueries({ queryKey: listKey }),
      ]),
  })
}