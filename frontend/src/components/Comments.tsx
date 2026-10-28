import { useState } from 'react'
import type { FormEvent, KeyboardEvent } from 'react'
import { ApiError } from '../lib/api'
import type { Comment } from '../lib/comments'
import { useComments, useCreateComment, useDeleteComment, useUpdateComment } from '../lib/comments'
import { can } from '../lib/permissions'
import type { Role } from '../lib/permissions'

const MAX_LENGTH = 5000

function when(iso: string): string {
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}

function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0].toUpperCase())
    .join('')
}

/** What the server said, when it gave a reason we can show. */
function reasonFor(err: unknown, fallback: string): string {
  return err instanceof ApiError && (err.status === 403 || err.status === 422) ? err.message : fallback
}

/** The discussion under a task: read it, add to it, and change your own comments. */
export function Comments({
  workspaceId,
  projectId,
  taskNumber,
  role,
  myUserId,
}: {
  workspaceId: string
  projectId: string
  taskNumber: number
  role: Role
  myUserId: string
}) {
  const { data: comments, isPending, isError, refetch } = useComments(workspaceId, projectId, taskNumber)
  const create = useCreateComment(workspaceId, projectId, taskNumber)
  const [draft, setDraft] = useState('')
  const [error, setError] = useState<string | null>(null)

  const canComment = can(role, 'comment:write')
  const canModerate = can(role, 'comment:moderate')

  async function post(event?: FormEvent) {
    event?.preventDefault()
    const body = draft.trim()
    if (!body || create.isPending) return
    if (body.length > MAX_LENGTH) {
      setError(`Comments can be at most ${MAX_LENGTH} characters.`)
      return
    }

    setError(null)
    try {
      await create.mutateAsync(body)
      setDraft('')
    } catch (err) {
      setError(reasonFor(err, 'Could not post the comment. Please try again.'))
    }
  }

  // Ctrl+Enter (or Cmd+Enter) posts without leaving the keyboard.
  function onKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) void post()
  }

  return (
    <section aria-label="Comments" className="mt-8 border-t border-slate-800 pt-6">
      <h3 className="font-semibold">Comments{comments && comments.length > 0 ? ` (${comments.length})` : ''}</h3>

      {isPending && <p className="mt-3 text-sm text-slate-400">Loading…</p>}
      {isError && (
        <p role="alert" className="mt-3 text-sm text-red-300">
          Could not load the comments.{' '}
          <button onClick={() => void refetch()} className="underline">
            Try again
          </button>
        </p>
      )}
      {comments && comments.length === 0 && <p className="mt-3 text-sm text-slate-400">No comments yet.</p>}

      {comments && comments.length > 0 && (
        <ul className="mt-4 space-y-5">
          {comments.map((comment) => (
            <CommentItem
              key={comment.id}
              comment={comment}
              workspaceId={workspaceId}
              projectId={projectId}
              taskNumber={taskNumber}
              isMine={comment.author_id !== null && comment.author_id === myUserId}
              canModerate={canModerate}
              canComment={canComment}
            />
          ))}
        </ul>
      )}

      {canComment ? (
        <form onSubmit={post} className="mt-5">
          <label htmlFor="new-comment" className="sr-only">
            Write a comment
          </label>
          <textarea
            id="new-comment"
            rows={3}
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={onKeyDown}
            placeholder="Write a comment…  (Ctrl+Enter to post)"
            className="w-full rounded-md bg-slate-800 px-3 py-2 text-sm text-white outline-none ring-1 ring-slate-600 focus:ring-2 focus:ring-indigo-400"
          />
          {error && (
            <p role="alert" className="mt-1 text-sm text-red-300">
              {error}
            </p>
          )}
          <button
            type="submit"
            disabled={create.isPending || !draft.trim()}
            className="mt-2 rounded-md bg-indigo-500 px-4 py-2 text-sm font-medium hover:bg-indigo-400 disabled:opacity-50"
          >
            {create.isPending ? 'Posting…' : 'Comment'}
          </button>
        </form>
      ) : (
        <p className="mt-5 text-sm text-slate-500">Viewers can read comments but not write them.</p>
      )}
    </section>
  )
}

function CommentItem({
  comment,
  workspaceId,
  projectId,
  taskNumber,
  isMine,
  canModerate,
  canComment,
}: {
  comment: Comment
  workspaceId: string
  projectId: string
  taskNumber: number
  isMine: boolean
  canModerate: boolean
  canComment: boolean
}) {
  const update = useUpdateComment(workspaceId, projectId, taskNumber)
  const remove = useDeleteComment(workspaceId, projectId, taskNumber)
  const [editing, setEditing] = useState(false)
  const [text, setText] = useState(comment.body)
  const [confirmingDelete, setConfirmingDelete] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const edited = new Date(comment.updated_at) > new Date(comment.created_at)
  const mayEdit = isMine && canComment // only the author, and only while they may still comment
  const mayDelete = isMine || canModerate // authors always; admins and owners moderate

  async function save() {
    const body = text.trim()
    if (!body || body.length > MAX_LENGTH) {
      setError(body ? `Comments can be at most ${MAX_LENGTH} characters.` : 'A comment cannot be empty.')
      return
    }
    if (body === comment.body) {
      setEditing(false)
      return
    }

    setError(null)
    try {
      await update.mutateAsync({ id: comment.id, body })
      setEditing(false)
    } catch (err) {
      setError(reasonFor(err, 'Could not save the change. Please try again.'))
    }
  }

  async function onDelete() {
    setError(null)
    try {
      await remove.mutateAsync(comment.id)
    } catch (err) {
      setError(reasonFor(err, 'Could not delete the comment. Please try again.'))
      setConfirmingDelete(false)
    }
  }

  return (
    <li className="flex gap-3">
      <span
        aria-hidden="true"
        className="mt-0.5 grid h-8 w-8 shrink-0 place-items-center rounded-full bg-indigo-500/30 text-xs font-medium"
      >
        {initials(comment.author_name)}
      </span>

      <div className="min-w-0 flex-1">
        <p className="text-sm">
          <span className="font-medium">{comment.author_name}</span>
          <span className="ml-2 text-xs text-slate-500">
            {when(comment.created_at)}
            {edited && ' · edited'}
          </span>
        </p>

        {editing ? (
          <div className="mt-1">
            <textarea
              aria-label="Edit comment"
              rows={3}
              autoFocus
              value={text}
              onChange={(event) => setText(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Escape') {
                  event.stopPropagation() // do not also close the task panel
                  setEditing(false)
                  setText(comment.body)
                } else if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
                  void save()
                }
              }}
              className="w-full rounded-md bg-slate-800 px-3 py-2 text-sm text-white outline-none ring-1 ring-slate-600 focus:ring-2 focus:ring-indigo-400"
            />
            <div className="mt-1 flex items-center gap-3 text-sm">
              <button
                onClick={() => void save()}
                disabled={update.isPending}
                className="rounded-md bg-indigo-500 px-3 py-1 font-medium hover:bg-indigo-400 disabled:opacity-60"
              >
                Save
              </button>
              <button
                onClick={() => {
                  setEditing(false)
                  setText(comment.body)
                  setError(null)
                }}
                className="text-slate-400 hover:text-white"
              >
                Cancel
              </button>
            </div>
          </div>
        ) : (
          <p className="mt-1 whitespace-pre-wrap break-words text-sm text-slate-200">{comment.body}</p>
        )}

        {error && (
          <p role="alert" className="mt-1 text-xs text-red-300">
            {error}
          </p>
        )}

        {!editing && (mayEdit || mayDelete) && (
          <p className="mt-1 flex items-center gap-3 text-xs text-slate-500">
            {mayEdit && (
              <button onClick={() => setEditing(true)} className="hover:text-white">
                Edit
              </button>
            )}
            {mayDelete &&
              (confirmingDelete ? (
                <span className="flex items-center gap-2">
                  <button
                    onClick={() => void onDelete()}
                    disabled={remove.isPending}
                    className="font-medium text-red-300 hover:text-red-200"
                  >
                    Delete this comment
                  </button>
                  <button onClick={() => setConfirmingDelete(false)} className="hover:text-white">
                    Cancel
                  </button>
                </span>
              ) : (
                <button onClick={() => setConfirmingDelete(true)} className="hover:text-white">
                  Delete
                </button>
              ))}
          </p>
        )}
      </div>
    </li>
  )
}