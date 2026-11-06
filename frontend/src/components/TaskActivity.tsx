import { useTaskActivity } from '../lib/activity'
import { ActivityList } from './ActivityList'

/** What happened to one task. It loads only when shown. */
export function TaskActivity({
  workspaceId,
  projectId,
  taskNumber,
}: {
  workspaceId: string
  projectId: string
  taskNumber: number
}) {
  const activity = useTaskActivity(workspaceId, projectId, taskNumber)

  if (activity.isPending) return <p className="text-sm text-slate-400">Loading…</p>
  if (activity.isError) {
    return (
      <p role="alert" className="text-sm text-red-300">
        Could not load the activity.{' '}
        <button onClick={() => void activity.refetch()} className="underline">
          Try again
        </button>
      </p>
    )
  }

  const events = activity.data.pages.flatMap((page) => page.events)

  return (
    <div>
      {events.length === 0 ? (
        <p className="text-sm text-slate-400">No activity recorded yet.</p>
      ) : (
        <ActivityList
          events={events}
          scope="task"
          hasMore={activity.hasNextPage}
          loadingMore={activity.isFetchingNextPage}
          onLoadMore={() => void activity.fetchNextPage()}
        />
      )}
      <button onClick={() => void activity.refetch()} className="mt-4 text-xs text-slate-500 hover:text-white">
        Refresh
      </button>
    </div>
  )
}