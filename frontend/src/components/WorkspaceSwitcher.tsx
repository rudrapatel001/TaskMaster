import { useMatch, useNavigate } from 'react-router-dom'
import { useWorkspaces } from '../lib/workspaces'

const CREATE_NEW = '__create__'

/** A dropdown in the header for moving between workspaces. */
export function WorkspaceSwitcher() {
  const { data: workspaces = [] } = useWorkspaces()
  const navigate = useNavigate()

  // The layout cannot see the child route's params, so read the URL directly.
  const match = useMatch('/w/:slug/*')
  const currentSlug = match?.params.slug
  const isKnown = workspaces.some((w) => w.slug === currentSlug)

  function onChange(value: string) {
    navigate(value === CREATE_NEW ? '/workspaces/new' : `/w/${value}`)
  }

  return (
    <div>
      <label htmlFor="workspace-switcher" className="sr-only">
        Workspace
      </label>
      <select
        id="workspace-switcher"
        value={isKnown ? currentSlug : ''}
        onChange={(event) => onChange(event.target.value)}
        className="rounded-md bg-slate-800 px-3 py-1.5 text-sm text-white ring-1 ring-slate-600 focus:ring-2 focus:ring-indigo-400"
      >
        {!isKnown && (
          <option value="" disabled>
            {workspaces.length === 0 ? 'No workspaces yet' : 'Choose a workspace'}
          </option>
        )}
        {workspaces.map((workspace) => (
          <option key={workspace.id} value={workspace.slug}>
            {workspace.name}
          </option>
        ))}
        <option value={CREATE_NEW}>+ Create workspace</option>
      </select>
    </div>
  )
}