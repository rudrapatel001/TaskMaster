import { useState } from 'react'
import { useAuth } from '../auth/AuthContext'
import { NameForm } from '../components/NameForm'
import { api, setAccessToken } from '../lib/api'
import type { User } from '../lib/api'

export function HomePage() {
  const { state, logout, updateUser } = useAuth()
  const [result, setResult] = useState<string | null>(null)

  if (state.status !== 'authenticated') return null
  const { user } = state

  async function loadProfile() {
    try {
      const me = await api<User>('/me')
      setResult(`GET /me worked: ${me.name} (${me.email})`)
    } catch {
      setResult('GET /me failed')
    }
  }

  return (
    <div className="min-h-screen bg-slate-900 px-4 py-12 text-white">
      <div className="mx-auto max-w-xl">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">TaskMaster</h1>
          <button
            onClick={logout}
            className="rounded-md bg-slate-700 px-4 py-2 text-sm hover:bg-slate-600"
          >
            Log out
          </button>
        </div>

        <p className="mt-8 text-xl">Hi, {user.name}</p>
        <p className="text-slate-400">{user.email}</p>

        <NameForm user={user} onSaved={updateUser} />

        {import.meta.env.DEV && (
          <div className="mt-10 rounded-xl border border-dashed border-slate-600 p-6">
            <h2 className="font-semibold">Developer tools (only shown in development)</h2>
            <p className="mt-1 text-sm text-slate-400">
              Break the access token, then load your profile. Watch the Network tab: the request fails
              with 401, a refresh happens, and the request is retried, all without you noticing.
            </p>
            <div className="mt-4 flex gap-3">
              <button
                onClick={() => setAccessToken('this-is-not-a-valid-token')}
                className="rounded-md bg-amber-500 px-3 py-2 text-sm font-medium text-slate-900 hover:bg-amber-400"
              >
                Break my access token
              </button>
              <button
                onClick={loadProfile}
                className="rounded-md bg-indigo-500 px-3 py-2 text-sm font-medium hover:bg-indigo-400"
              >
                Load my profile
              </button>
            </div>
            {result && <p className="mt-4 text-sm text-slate-300">{result}</p>}
          </div>
        )}
      </div>
    </div>
  )
}