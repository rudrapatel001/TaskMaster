import type { ComponentProps } from 'react'

interface TextFieldProps extends ComponentProps<'input'> {
  id: string
  label: string
  error?: string
}

/** A labelled text input with an error message underneath. */
export function TextField({ id, label, error, ...inputProps }: TextFieldProps) {
  return (
    <div className="mt-4">
      <label htmlFor={id} className="block text-sm text-slate-300">
        {label}
      </label>
      <input
        id={id}
        aria-invalid={error ? true : undefined}
        className="mt-1 w-full rounded-md bg-slate-900 px-3 py-2 text-white outline-none ring-1 ring-slate-600 focus:ring-2 focus:ring-indigo-400"
        {...inputProps}
      />
      {error && <p className="mt-1 text-sm text-red-300">{error}</p>}
    </div>
  )
}