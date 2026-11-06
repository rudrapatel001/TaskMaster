import { ActivityList } from '../components/ActivityList'
import { QueryError } from '../components/QueryError'
import { WorkspaceGate } from '../components/WorkspaceGate'
import { WorkspaceHeader } from '../components/WorkspaceHeader'
import { useWorkspaceActivity } from '../lib/activity'
import type { Workspace } from '../lib/workspaces'

export function ActivityPage() {
  return <WorkspaceGate>{(workspace) => <Activity workspace={workspace} />}</WorkspaceGate>
}

function Activity({ workspace }: { workspace: Workspace }) {
  const activity = useWorkspaceActivity(workspace.id)

  return (
    <div>
      <WorkspaceHeader workspace={workspace} />
      <h2 className="mt-8 text-xl font-semibold">Activity</h2>

      {activity.isPending && <p className="mt-4 text-slate-400">Loading…</p>}
      {activity.isError && (
        <div className="mt-4">
          <QueryError message="Could not load the activity." onRetry={() => void activity.refetch()} />
        </div>
      )}

      {activity.data && (
        <div className="mt-4">
          {activity.data.pages[0].events.length === 0 ? (
            <p className="text-slate-400">Nothing has happened yet. Create or move a task and it will show up here.</p>
          ) : (
            <ActivityList
              events={activity.data.pages.flatMap((page) => page.events)}
              scope="workspace"
              workspaceSlug={workspace.slug}
              hasMore={activity.hasNextPage}
              loadingMore={activity.isFetchingNextPage}
              onLoadMore={() => void activity.fetchNextPage()}
            />
          )}
        </div>
      )}
    </div>
  )
}