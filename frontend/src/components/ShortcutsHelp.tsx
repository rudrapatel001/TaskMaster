import { useEffect, useRef } from 'react'
import type { ReactNode } from 'react'

const isMac = typeof navigator !== 'undefined' && navigator.userAgent.includes('Mac')
const modifier = isMac ? '⌘' : 'Ctrl'

function Keys({ children }: { children: ReactNode }) {
  return <span className="flex shrink-0 items-center gap-1">{children}</span>
}

function Key({ children }: { children: ReactNode }) {
  return (
    <kbd className="rounded border border-slate-600 bg-slate-900 px-1.5 py-0.5 font-mono text-xs text-slate-200">
      {children}
    </kbd>
  )
}

const shortcuts: { keys: ReactNode; does: string }[] = [
  {
    keys: (
      <Keys>
        <Key>{modifier}</Key>
        <Key>K</Key>
        <span className="text-slate-500">or</span>
        <Key>/</Key>
      </Keys>
    ),
    does: 'Open the command palette: jump anywhere, run an action, or find a task by title or ID',
  },
  { keys: <Key>c</Key>, does: 'Add a task (on a board)' },
  {
    keys: (
      <Keys>
        <Key>g</Key>
        <span className="text-slate-500">then</span>
        <Key>o</Key>
      </Keys>
    ),
    does: 'Go to the workspace Overview',
  },
  {
    keys: (
      <Keys>
        <Key>g</Key>
        <span className="text-slate-500">then</span>
        <Key>p</Key>
      </Keys>
    ),
    does: 'Go to Projects',
  },
  {
    keys: (
      <Keys>
        <Key>g</Key>
        <span className="text-slate-500">then</span>
        <Key>a</Key>
      </Keys>
    ),
    does: 'Go to Activity',
  },
  {
    keys: (
      <Keys>
        <Key>g</Key>
        <span className="text-slate-500">then</span>
        <Key>m</Key>
      </Keys>
    ),
    does: 'Go to Members',
  },
  { keys: <Key>?</Key>, does: 'Show this list' },
  { keys: <Key>Esc</Key>, does: 'Close a dialog, a panel, or the palette' },
  {
    keys: (
      <Keys>
        <Key>Enter</Key>
      </Keys>
    ),
    does: 'On a board: open the focused card',
  },
  {
    keys: (
      <Keys>
        <Key>Space</Key>
        <span className="text-slate-500">then</span>
        <Key>↑ ↓ ← →</Key>
      </Keys>
    ),
    does: 'On a board: pick up the focused card, move it, and Space again to drop it',
  },
]

/** The keyboard cheat sheet. */
export function ShortcutsHelp({ onClose }: { onClose: () => void }) {
  const dialogRef = useRef<HTMLDivElement>(null)

  // Focus the dialog, give focus back when it closes, and close on Escape.
  useEffect(() => {
    const before = document.activeElement
    dialogRef.current?.focus()

    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      if (before instanceof HTMLElement) before.focus()
    }
  }, [onClose])

  return (
    <div
      className="fixed inset-0 z-40 flex items-start justify-center bg-black/60 px-4 pt-[10vh]"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose()
      }}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="shortcuts-title"
        tabIndex={-1}
        className="w-full max-w-lg rounded-xl bg-slate-800 p-6 shadow-2xl ring-1 ring-slate-700 focus:outline-none"
      >
        <div className="flex items-center justify-between">
          <h2 id="shortcuts-title" className="font-semibold">
            Keyboard shortcuts
          </h2>
          <button
            onClick={onClose}
            aria-label="Close"
            className="rounded-md px-2 py-1 text-slate-400 hover:bg-slate-700 hover:text-white"
          >
            ✕
          </button>
        </div>
        <p className="mt-1 text-sm text-slate-400">Single-key shortcuts work anywhere you are not typing in a box.</p>

        <ul className="mt-4 space-y-3">
          {shortcuts.map((shortcut, index) => (
            <li key={index} className="flex items-start justify-between gap-4 text-sm">
              <span className="text-slate-300">{shortcut.does}</span>
              {shortcut.keys}
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}