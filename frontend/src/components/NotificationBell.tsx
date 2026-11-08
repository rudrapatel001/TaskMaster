import { useEffect, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import type { AppNotification } from '../lib/notifications'
import { useMarkAllRead, useMarkRead, useNotifications, useUnreadCount } from '../lib/notifications'

function text(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

function when(iso: string): string {
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}

/** Where clicking a notification goes: the task, open on its board. */
function linkFor(notification: AppNotification): string | null {
  const slug = text(notification.metadata.workspace_slug)
  const match = /^([A-Z][A-Z0-9]*)-(\d+)$/.exec(text(notification.metadata.identifier))
  if (!slug || !match) return null
  return `/w/${slug}/projects/${match[1]}?task=${match[2]}`
}

/** The sentence after the person's name. */
function describe(notification: AppNotification): ReactNode {
  const task = (
    <>
      <strong className="font-medium">{text(notification.metadata.identifier)}</strong> {text(notification.metadata.title)}
    </>
  )

  switch (notification.type) {
    case 'task.assigned':
      return <>assigned you {task}</>
    case 'task.commented': {
      const excerpt = text(notification.metadata.excerpt)
      return (
        <>
          commented on {task}
          {excerpt && <span className="text-slate-400">: “{excerpt}”</span>}
        </>
      )
    }
    default:
      return <>{notification.type}</> // a kind this version does not know yet
  }
}

/** The bell in the header: a number of unread notifications, and a list that opens under it. */
export function NotificationBell() {
  const [open, setOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const { data: unread = 0 } = useUnreadCount()

  // While open: close on Escape, or on a click anywhere outside.
  useEffect(() => {
    if (!open) return

    function onMouseDown(event: MouseEvent) {
      if (!containerRef.current?.contains(event.target as Node)) setOpen(false)
    }
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') setOpen(false)
    }

    document.addEventListener('mousedown', onMouseDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('mousedown', onMouseDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  return (
    <div ref={containerRef} className="relative">
      <button
        onClick={() => setOpen((current) => !current)}
        aria-label={unread > 0 ? `Notifications, ${unread} unread` : 'Notifications'}
        aria-haspopup="true"
        aria-expanded={open}
        className="relative rounded-md p-2 text-slate-300 hover:bg-slate-800 hover:text-white"
      >
        <svg
          viewBox="0 0 24 24"
          width="20"
          height="20"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <path d="M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9" />
          <path d="M13.73 21a2 2 0 0 1-3.46 0" />
        </svg>
        {unread > 0 && (
          <span
            aria-hidden="true"
            className="absolute -right-0.5 -top-0.5 grid h-4 min-w-4 place-items-center rounded-full bg-red-500 px-1 text-[10px] font-medium text-white"
          >
            {unread > 99 ? '99+' : unread}
          </span>
        )}
      </button>

      {open && <NotificationPanel unread={unread} onClose={() => setOpen(false)} />}
    </div>
  )
}

function NotificationPanel({ unread, onClose }: { unread: number; onClose: () => void }) {
  const navigate = useNavigate()
  const list = useNotifications(true)
  const markRead = useMarkRead()
  const markAllRead = useMarkAllRead()

  const items = list.data?.pages.flatMap((page) => page.notifications) ?? []

  function openNotification(notification: AppNotification) {
    if (notification.read_at === null) markRead.mutate(notification.id)
    onClose()
    const target = linkFor(notification)
    if (target) navigate(target)
  }

  return (
    <div
      role="region"
      aria-label="Notifications"
      className="absolute right-0 z-30 mt-2 w-80 overflow-hidden rounded-xl bg-slate-800 shadow-2xl ring-1 ring-slate-700"
    >
      <div className="flex items-center justify-between border-b border-slate-700 px-4 py-3">
        <h2 className="text-sm font-semibold">Notifications</h2>
        {unread > 0 && (
          <button onClick={() => markAllRead.mutate()} className="text-xs text-indigo-300 hover:underline">
            Mark all as read
          </button>
        )}
      </div>

      <div className="max-h-96 overflow-y-auto">
        {list.isPending && <p className="px-4 py-6 text-sm text-slate-400">Loading…</p>}
        {list.isError && (
          <p role="alert" className="px-4 py-6 text-sm text-red-300">
            Could not load your notifications.{' '}
            <button onClick={() => void list.refetch()} className="underline">
              Try again
            </button>
          </p>
        )}
        {list.data && items.length === 0 && (
          <p className="px-4 py-6 text-sm text-slate-400">You're all caught up.</p>
        )}

        {items.length > 0 && (
          <ul className="divide-y divide-slate-700">
            {items.map((notification) => {
              const isUnread = notification.read_at === null
              return (
                <li key={notification.id}>
                  <button
                    onClick={() => openNotification(notification)}
                    className={`block w-full px-4 py-3 text-left text-sm hover:bg-slate-700 ${
                      isUnread ? 'bg-slate-700/40' : ''
                    }`}
                  >
                    <span className="flex gap-2">
                      <span
                        aria-label={isUnread ? 'Unread' : undefined}
                        className={`mt-1.5 h-2 w-2 shrink-0 rounded-full ${isUnread ? 'bg-indigo-400' : 'bg-transparent'}`}
                      />
                      <span>
                        <span className="font-medium">{notification.actor_name}</span> {describe(notification)}
                        <span className="mt-1 block text-xs text-slate-500">{when(notification.created_at)}</span>
                      </span>
                    </span>
                  </button>
                </li>
              )
            })}
          </ul>
        )}

        {list.hasNextPage && (
          <button
            onClick={() => void list.fetchNextPage()}
            disabled={list.isFetchingNextPage}
            className="w-full border-t border-slate-700 px-4 py-3 text-sm text-slate-300 hover:bg-slate-700 disabled:opacity-60"
          >
            {list.isFetchingNextPage ? 'Loading…' : 'Load older'}
          </button>
        )}
      </div>
    </div>
  )
}