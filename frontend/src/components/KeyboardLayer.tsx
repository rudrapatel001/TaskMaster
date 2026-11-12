import { useCallback, useState } from 'react'
import { useMatch, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import type { Command } from '../lib/commands'
import { can } from '../lib/permissions'
import { useProjects } from '../lib/projects'
import { ADD_TASK_EVENT, useGlobalShortcuts } from '../lib/shortcuts'
import type { ShortcutAction } from '../lib/shortcuts'
import { useWorkspaces } from '../lib/workspaces'
import type { Workspace } from '../lib/workspaces'
import { CommandPalette } from './CommandPalette'
import { ShortcutsHelp } from './ShortcutsHelp'

const isMac = typeof navigator !== 'undefined' && navigator.userAgent.includes('Mac')

/**
 * Everything keyboard-driven: the shortcuts, the command palette, the cheat
 * sheet, and the "Search" button in the header that opens the palette.
 */
export function KeyboardLayer() {
  const navigate = useNavigate()
  const workspaceMatch = useMatch('/w/:slug/*')
  const boardMatch = useMatch('/w/:slug/projects/:key')
  const workspaces = useWorkspaces()

  const slug = workspaceMatch?.params.slug
  const workspace = workspaces.data?.find((w) => w.slug === slug)
  const onBoard = boardMatch !== null
  const canAddTask = onBoard && can(workspace?.role, 'task:write')

  const [paletteOpen, setPaletteOpen] = useState(false)
  const [helpOpen, setHelpOpen] = useState(false)
  const anyOpen = paletteOpen || helpOpen

  const onAction = useCallback(
    (action: ShortcutAction) => {
      if (action === 'palette') {
        setHelpOpen(false)
        setPaletteOpen((open) => !open)
        return
      }
      if (action === 'help') {
        setPaletteOpen(false)
        setHelpOpen((open) => !open)
        return
      }
      if (anyOpen) return // single-key shortcuts do nothing behind a dialog

      if (action === 'new-task') {
        if (canAddTask) window.dispatchEvent(new Event(ADD_TASK_EVENT))
        return
      }
      if (!slug) return
      const pages: Partial<Record<ShortcutAction, string>> = {
        'go-overview': `/w/${slug}`,
        'go-projects': `/w/${slug}/projects`,
        'go-activity': `/w/${slug}/activity`,
        'go-members': `/w/${slug}/members`,
      }
      const target = pages[action]
      if (target) navigate(target)
    },
    [anyOpen, canAddTask, navigate, slug],
  )
  useGlobalShortcuts(onAction)

  return (
    <>
      <button
        onClick={() => setPaletteOpen(true)}
        aria-label="Open the command palette"
        aria-keyshortcuts="Control+K Meta+K"
        className="flex items-center gap-2 rounded-md bg-slate-800 px-3 py-1.5 text-sm text-slate-400 ring-1 ring-slate-700 hover:text-white"
      >
        Search
        <kbd className="rounded border border-slate-600 px-1 font-mono text-xs">{isMac ? '⌘K' : 'Ctrl K'}</kbd>
      </button>

      {paletteOpen && (
        <PaletteHost
          workspace={workspace}
          boardKey={boardMatch?.params.key}
          canAddTask={canAddTask}
          onClose={() => setPaletteOpen(false)}
          onOpenHelp={() => setHelpOpen(true)}
        />
      )}
      {helpOpen && <ShortcutsHelp onClose={() => setHelpOpen(false)} />}
    </>
  )
}

interface HostProps {
  workspace: Workspace | undefined
  boardKey: string | undefined
  canAddTask: boolean
  onClose: () => void
  onOpenHelp: () => void
}

/** Chooses what the palette can offer: with a workspace in the URL it also knows its projects. */
function PaletteHost(props: HostProps) {
  return props.workspace ? (
    <WorkspacePalette {...props} workspace={props.workspace} />
  ) : (
    <CommandPaletteWith {...props} projectCommands={[]} />
  )
}

function WorkspacePalette(props: HostProps & { workspace: Workspace }) {
  const navigate = useNavigate()
  const projects = useProjects(props.workspace.id) // fetched when the palette opens, then cached

  const projectCommands: Command[] = (projects.data ?? []).map((project) => ({
    id: `project-${project.id}`,
    title: `${project.key} · ${project.name}`,
    group: 'Projects',
    keywords: 'project board open',
    run: () => navigate(`/w/${props.workspace.slug}/projects/${project.key}`),
  }))

  return <CommandPaletteWith {...props} projectCommands={projectCommands} />
}

function CommandPaletteWith({
  workspace,
  boardKey,
  canAddTask,
  onClose,
  onOpenHelp,
  projectCommands,
}: HostProps & { projectCommands: Command[] }) {
  const navigate = useNavigate()
  const { logout } = useAuth()
  const workspaces = useWorkspaces()
  const slug = workspace?.slug

  const commands: Command[] = []

  // Actions
  if (canAddTask) {
    commands.push({
      id: 'add-task',
      title: 'Add a task to this board',
      group: 'Actions',
      hint: 'c',
      keywords: 'new create task',
      run: () => window.dispatchEvent(new Event(ADD_TASK_EVENT)),
    })
  }
  if (workspace && boardKey && can(workspace.role, 'project:manage')) {
    commands.push({
      id: 'project-settings',
      title: 'Project settings',
      group: 'Actions',
      keywords: 'columns rename delete configure',
      run: () => navigate(`/w/${workspace.slug}/projects/${boardKey}/settings`),
    })
  }
  commands.push({
    id: 'create-workspace',
    title: 'Create a workspace',
    group: 'Actions',
    keywords: 'new team',
    run: () => navigate('/workspaces/new'),
  })
  commands.push({
    id: 'shortcuts',
    title: 'Keyboard shortcuts',
    group: 'Actions',
    hint: '?',
    keywords: 'help keys cheat sheet',
    run: onOpenHelp,
  })
  commands.push({ id: 'log-out', title: 'Log out', group: 'Actions', keywords: 'sign out', run: () => void logout() })

  // Go to
  if (slug) {
    commands.push(
      { id: 'go-overview', title: 'Overview', group: 'Go to', hint: 'g o', run: () => navigate(`/w/${slug}`) },
      { id: 'go-projects', title: 'Projects', group: 'Go to', hint: 'g p', run: () => navigate(`/w/${slug}/projects`) },
      { id: 'go-activity', title: 'Activity', group: 'Go to', hint: 'g a', run: () => navigate(`/w/${slug}/activity`) },
      { id: 'go-members', title: 'Members', group: 'Go to', hint: 'g m', run: () => navigate(`/w/${slug}/members`) },
    )
  }
  commands.push({ id: 'go-profile', title: 'Your profile', group: 'Go to', run: () => navigate('/profile') })

  commands.push(...projectCommands)

  // Workspaces
  for (const other of workspaces.data ?? []) {
    if (other.slug === slug) continue
    commands.push({
      id: `workspace-${other.id}`,
      title: `Switch to ${other.name}`,
      group: 'Workspaces',
      keywords: 'workspace team',
      run: () => navigate(`/w/${other.slug}`),
    })
  }

  return <CommandPalette commands={commands} workspaceId={workspace?.id} workspaceSlug={slug} onClose={onClose} />
}