import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from './AuthContext'

/** Pages inside this route are only shown to logged-in users. */
export function ProtectedRoute() {
  const { state } = useAuth()
  const location = useLocation()

  if (state.status === 'loading') {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-900 text-slate-400">
        Loading…
      </div>
    )
  }

  if (state.status === 'anonymous') {
    // Remember where the visitor was going, so login can send them back.
    return <Navigate to="/login" replace state={{ from: location }} />
  }

  return <Outlet />
}