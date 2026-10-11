import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { api, refreshSession, setAccessToken } from '../lib/api'
import type { Session } from '../lib/api'
import { AuthContext } from './AuthContext'
import type { AuthContextValue, AuthState } from './AuthContext'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({ status: 'loading' })

  // On page load there is no access token in memory, but the browser may still
  // hold a refresh cookie. Try to turn it into a session.
  useEffect(() => {
    let cancelled = false

    refreshSession().then((session) => {
      if (cancelled) return
      setState(session ? { status: 'authenticated', user: session.user } : { status: 'anonymous' })
    })

    return () => {
      cancelled = true
    }
  }, [])

  const login = useCallback(async (email: string, password: string) => {
    const session = await api<Session>('/auth/login', {
      method: 'POST',
      body: { email, password },
      auth: false,
    })
    setAccessToken(session.access_token)
    setState({ status: 'authenticated', user: session.user })
  }, [])

  const logout = useCallback(async () => {
    try {
      await api<void>('/auth/logout', { method: 'POST', auth: false })
    } catch {
      // Even if the server cannot be reached, log out locally.
    }
    setAccessToken(null)
    setState({ status: 'anonymous' })
  }, [])

  const value = useMemo<AuthContextValue>(() => ({ state, login, logout }), [state, login, logout])

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}