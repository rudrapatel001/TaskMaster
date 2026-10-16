/** Shown when data could not be loaded, with a button to try again. */
export function QueryError({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div role="alert" className="rounded-md bg-red-500/10 px-4 py-3 text-sm text-red-300">
      <p>{message}</p>
      <button onClick={onRetry} className="mt-2 rounded-md bg-slate-700 px-3 py-1.5 text-white hover:bg-slate-600">
        Try again
      </button>
    </div>
  )
}