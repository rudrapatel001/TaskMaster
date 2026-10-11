import { createContext, useContext } from 'react'
import type { User } from '../lib/api'

export type AuthState =
  | { status: 'loading' } // still checking whether the visitor has a session
  | { status: 'anonymous' } // not logged in
  | { status: 'authenticated'; user: User }

export interface AuthContextValue {
  state: AuthState
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

export const AuthContext = createContext<AuthContextValue | null>(null)

export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext)
  if (!value) throw new Error('useAuth must be used inside <AuthProvider>')
  return value
}