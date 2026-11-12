import { useEffect, useId, useMemo, useRef, useState } from 'react'
import type { KeyboardEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import type { Command } from '../lib/commands'
import { filterCommands } from '../lib/commands'
import { useTaskSearch } from '../lib/search'
import { useDebounced } from '../lib/useDebounced'

/**
 * The command palette: type to find a page, project, workspace, action or task,
 * move with the arrow keys, press Enter to run it and Escape to close.
 */
export function CommandPalette({
  commands,
  workspaceId,
  workspaceSlug,
  onClose,
}: {
  commands: Command[] // everything except tasks, which are searched on the server
  workspaceId?: string
  workspaceSlug?: string
  onClose: () => void
}) {
  const navigate = useNavigate()
  const inputRef = useRef<HTMLInputElement>(null)
  const listId = useId()
  const [query, setQuery] = useState('')
  const [active, setActive] = useState(0)

  // Wait for a pause in typing before asking the server about tasks.
  const tasks = useTaskSearch(workspaceId, useDebounced(query, 200))

  const results = useMemo(() => {
    const taskCommands: Command[] = (tasks.data ?? []).map((task) => ({
      id: `task-${task.identifier}`,
      title: `${task.identifier} ${task.title}`,
      group: 'Tasks',
      hint: task.project_name,
      run: () => navigate(`/w/${workspaceSlug}/projects/${task.project_key}?task=${task.number}`),
    }))
    return [...filterCommands(commands, query), ...taskCommands]
  }, [commands, query, tasks.data, navigate, workspaceSlug])

  const activeIndex = Math.min(active, results.length - 1)
  const optionId = (index: number) => `${listId}-option-${index}`
  const searching = query.trim().length >= 2 && workspaceId !== undefined

  // Put the cursor in the box, and give focus back to where it was when the palette closes.
  useEffect(() => {
    const before = document.activeElement
    inputRef.current?.focus()
    return () => {
      if (before instanceof HTMLElement) before.focus()
    }
  }, [])

  // Keep the highlighted result in view when moving with the arrow keys.
  useEffect(() => {
    document.getElementById(`${listId}-option-${activeIndex}`)?.scrollIntoView({ block: 'nearest' })
  }, [activeIndex, listId])

  function choose(command: Command) {
    onClose()
    // After the palette has gone, so that focus lands where the command wants it.
    window.setTimeout(command.run, 0)
  }

  function onKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    switch (event.key) {
      case 'ArrowDown':
        event.preventDefault()
        if (results.length > 0) setActive((activeIndex + 1) % results.length)
        break
      case 'ArrowUp':
        event.preventDefault()
        if (results.length > 0) setActive((activeIndex - 1 + results.length) % results.length)
        break
      case 'Enter':
        event.preventDefault()
        if (results[activeIndex]) choose(results[activeIndex])
        break
      case 'Escape':
        event.stopPropagation() // do not also close a task panel behind
        onClose()
        break
      case 'Tab':
        event.preventDefault() // keep focus in the box
        break
    }
  }

  // With nothing typed, results are shown under headings; typing sorts them by relevance instead.
  const showHeadings = query.trim() === ''

  return (
    <div
      className="fixed inset-0 z-40 flex items-start justify-center bg-black/60 px-4 pt-[15vh]"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose()
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Command palette"
        className="w-full max-w-xl overflow-hidden rounded-xl bg-slate-800 shadow-2xl ring-1 ring-slate-700"
      >
        <input
          ref={inputRef}
          role="combobox"
          aria-expanded="true"
          aria-controls={listId}
          aria-autocomplete="list"
          aria-activedescendant={results.length > 0 ? optionId(activeIndex) : undefined}
          aria-label="Search commands, pages, projects and tasks"
          value={query}
          onChange={(event) => {
            setQuery(event.target.value)
            setActive(0)
          }}
          onKeyDown={onKeyDown}
          placeholder="Type a command, a project, or a task (try WEB-12)…"
          autoComplete="off"
          className="w-full border-b border-slate-700 bg-transparent px-4 py-3 text-white outline-none placeholder:text-slate-500"
        />

        <ul id={listId} role="listbox" aria-label="Results" className="max-h-80 overflow-y-auto py-2">
          {results.map((command, index) => {
            const heading = showHeadings && command.group !== results[index - 1]?.group
            return (
              <li key={command.id} role="presentation">
                {heading && (
                  <p className="px-4 pb-1 pt-3 text-xs font-medium uppercase tracking-wide text-slate-500">
                    {command.group}
                  </p>
                )}
                <div
                  id={optionId(index)}
                  role="option"
                  aria-selected={index === activeIndex}
                  onMouseMove={() => setActive(index)}
                  onClick={() => choose(command)}
                  className={`flex cursor-pointer items-center justify-between gap-4 px-4 py-2 text-sm ${
                    index === activeIndex ? 'bg-indigo-500/20 text-white' : 'text-slate-300'
                  }`}
                >
                  <span className="truncate">{command.title}</span>
                  <span className="shrink-0 text-xs text-slate-500">
                    {command.hint ?? (showHeadings ? '' : command.group)}
                  </span>
                </div>
              </li>
            )
          })}
        </ul>

        <p aria-live="polite" className="border-t border-slate-700 px-4 py-2 text-xs text-slate-500">
          {tasks.isError
            ? 'Could not search tasks.'
            : searching && tasks.isFetching
              ? 'Searching tasks…'
              : results.length === 0
                ? 'No matches.'
                : '↑ ↓ to move · Enter to choose · Esc to close'}
        </p>
      </div>
    </div>
  )
}