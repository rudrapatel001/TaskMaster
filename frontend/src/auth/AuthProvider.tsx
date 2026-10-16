import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { api, refreshSession, setAccessToken, setSessionExpiredHandler } from '../lib/api'
import type { Session, User } from '../lib/api'
import { AuthContext } from './AuthContext'
import type { AuthContextValue, AuthState } from './AuthContext'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({ status: 'loading' })
  const queryClient = useQueryClient()

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

  // If the session ends while the app is open (the refresh token expired, or
  // was revoked), go back to the login page and forget everything we loaded.
  useEffect(() => {
    setSessionExpiredHandler(() => {
      setAccessToken(null)
      queryClient.clear()
      setState({ status: 'anonymous' })
    })
    return () => setSessionExpiredHandler(null)
  }, [queryClient])

  const login = useCallback(
    async (email: string, password: string) => {
      const session = await api<Session>('/auth/login', {
        method: 'POST',
        body: { email, password },
        auth: false,
      })
      setAccessToken(session.access_token)
      queryClient.clear() // never show a previous user's data
      setState({ status: 'authenticated', user: session.user })
    },
    [queryClient],
  )

  const logout = useCallback(async () => {
    try {
      await api<void>('/auth/logout', { method: 'POST', auth: false })
    } catch {
      // Even if the server cannot be reached, log out locally.
    }
    setAccessToken(null)
    // Stop anything still loading, then forget all cached data, so the next
    // person to log in on this browser cannot see this user's workspaces.
    await queryClient.cancelQueries()
    queryClient.clear()
    setState({ status: 'anonymous', reason: 'logout' })
  }, [queryClient])

  const updateUser = useCallback((user: User) => {
    setState((current) => (current.status === 'authenticated' ? { status: 'authenticated', user } : current))
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({ state, login, logout, updateUser }),
    [state, login, logout, updateUser],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}