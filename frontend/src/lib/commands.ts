/** One thing the command palette can do. */
export interface Command {
  id: string
  title: string
  group: string // the heading it appears under, such as "Actions" or "Projects"
  hint?: string // small text on the right, such as a shortcut
  keywords?: string // extra words that should find it
  run: () => void
}

/** Where a typed word was found in a command: the earlier and cleaner, the higher the score. */
function scoreWord(text: string, word: string): number {
  const at = text.indexOf(word)
  if (at === -1) return 0
  if (at === 0) return 3 // starts the title
  if (text[at - 1] === ' ') return 2 // starts a word
  return 1 // somewhere inside
}

/**
 * Filters commands by what was typed and puts the best matches first.
 *
 * Every word typed must appear in the command (its title, or its keywords), so
 * "new task" finds "Add a new task". With nothing typed, everything is returned
 * in its original order. Among equal scores the original order is kept.
 */
export function filterCommands(commands: Command[], query: string): Command[] {
  const words = query.toLowerCase().split(/\s+/).filter(Boolean)
  if (words.length === 0) return commands

  const scored: { command: Command; score: number; index: number }[] = []
  commands.forEach((command, index) => {
    const title = command.title.toLowerCase()
    const everything = `${title} ${(command.keywords ?? '').toLowerCase()}`

    let score = 0
    for (const word of words) {
      const wordScore = Math.max(scoreWord(title, word), scoreWord(everything, word) > 0 ? 1 : 0)
      if (wordScore === 0) return // this word is nowhere in the command: leave it out
      score += wordScore
    }
    scored.push({ command, score, index })
  })

  return scored.sort((a, b) => b.score - a.score || a.index - b.index).map((entry) => entry.command)
}