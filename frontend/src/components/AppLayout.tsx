import { Link, Outlet, useMatch } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { NotificationBell } from './NotificationBell'
import { WorkspaceSwitcher } from './WorkspaceSwitcher'

/** The frame around every logged-in page: a header with navigation, then the page. */
export function AppLayout() {
  const { state, logout } = useAuth()
  const name = state.status === 'authenticated' ? state.user.name : ''
  const isBoard = useMatch('/w/:slug/projects/:key') !== null // boards need the full width

  return (
    <div className="min-h-screen bg-slate-900 text-white">
      <header className="border-b border-slate-800">
        <nav aria-label="Main" className="mx-auto flex max-w-3xl items-center gap-4 px-4 py-3">
          <Link to="/" className="font-bold">
            TaskMaster
          </Link>
          <WorkspaceSwitcher />
          <div className="ml-auto flex items-center gap-4 text-sm">
            <NotificationBell />
            <Link to="/profile" className="text-slate-300 hover:text-white">
              {name}
            </Link>
            <button onClick={logout} className="rounded-md bg-slate-700 px-3 py-1.5 hover:bg-slate-600">
              Log out
            </button>
          </div>
        </nav>
      </header>

      <main className={`mx-auto px-4 py-10 ${isBoard ? 'max-w-none' : 'max-w-3xl'}`}>
        <Outlet />
      </main>
    </div>
  )
}