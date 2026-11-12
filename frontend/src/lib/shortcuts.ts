import { useEffect } from 'react'

/** Fired when "c" is pressed on a board; the first column's "+ Add task" box listens for it. */
export const ADD_TASK_EVENT = 'taskmaster:add-task'

export type ShortcutAction =
  | 'palette' // open the command palette
  | 'help' // show the cheat sheet
  | 'new-task' // add a task on the board
  | 'go-overview'
  | 'go-projects'
  | 'go-activity'
  | 'go-members'

/** What was pressed, and whether the person is typing in a text box. */
export interface KeyInfo {
  key: string
  ctrl: boolean
  meta: boolean
  alt: boolean
  typing: boolean
}

/** What the shortcut layer remembers between key presses: just whether "g" was pressed. */
export interface ShortcutState {
  pendingG: boolean
}

export const initialShortcutState: ShortcutState = { pendingG: false }

const goTargets: Record<string, ShortcutAction> = {
  o: 'go-overview',
  p: 'go-projects',
  a: 'go-activity',
  m: 'go-members',
}

/**
 * Decides what a key press means. It is a plain function (no React, no
 * browser), so it is easy to test: given the key and what was pending, it
 * returns the action to run (if any), the new pending state, and whether the
 * browser's own handling of the key should be cancelled.
 *
 * - Ctrl/Cmd+K opens the palette, even in a text box.
 * - Every other shortcut is ignored while typing, and with Ctrl, Cmd or Alt held.
 * - "g" then o / p / a / m goes to Overview / Projects / Activity / Members.
 */
export function interpretKey(
  state: ShortcutState,
  k: KeyInfo,
): { action: ShortcutAction | null; state: ShortcutState; preventDefault: boolean } {
  if ((k.ctrl || k.meta) && !k.alt && k.key.toLowerCase() === 'k') {
    return { action: 'palette', state: initialShortcutState, preventDefault: true }
  }

  if (k.typing || k.ctrl || k.meta || k.alt) {
    return { action: null, state: initialShortcutState, preventDefault: false }
  }

  if (state.pendingG) {
    const action = goTargets[k.key] ?? null
    return { action, state: initialShortcutState, preventDefault: action !== null }
  }

  switch (k.key) {
    case 'g':
      return { action: null, state: { pendingG: true }, preventDefault: false }
    case '?':
      return { action: 'help', state: initialShortcutState, preventDefault: true }
    case '/':
      return { action: 'palette', state: initialShortcutState, preventDefault: true }
    case 'c':
      return { action: 'new-task', state: initialShortcutState, preventDefault: true }
    default:
      return { action: null, state: initialShortcutState, preventDefault: false }
  }
}

/** True when the keyboard focus is somewhere text is typed. */
export function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false
  return (
    target.isContentEditable ||
    target.tagName === 'INPUT' ||
    target.tagName === 'TEXTAREA' ||
    target.tagName === 'SELECT'
  )
}

const SEQUENCE_TIMEOUT_MS = 1500

/** Listens for the shortcuts above for as long as the component is mounted. */
export function useGlobalShortcuts(onAction: (action: ShortcutAction) => void): void {
  useEffect(() => {
    let state = initialShortcutState
    let timer: ReturnType<typeof setTimeout> | undefined

    function onKeyDown(event: KeyboardEvent) {
      const result = interpretKey(state, {
        key: event.key,
        ctrl: event.ctrlKey,
        meta: event.metaKey,
        alt: event.altKey,
        typing: isTypingTarget(event.target),
      })

      state = result.state
      clearTimeout(timer)
      // "g" waits a moment for its second key, then is forgotten.
      if (state.pendingG) timer = setTimeout(() => (state = initialShortcutState), SEQUENCE_TIMEOUT_MS)

      if (result.preventDefault) event.preventDefault()
      if (result.action) onAction(result.action)
    }

    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      clearTimeout(timer)
    }
  }, [onAction])
}