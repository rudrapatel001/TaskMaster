import { Navigate, Route, Routes } from 'react-router-dom'
import { ProtectedRoute } from './auth/ProtectedRoute'
import { AppLayout } from './components/AppLayout'
import { AcceptInvitePage } from './pages/AcceptInvitePage'
import { CreateWorkspacePage } from './pages/CreateWorkspacePage'
import { LoginPage } from './pages/LoginPage'
import { MembersPage } from './pages/MembersPage'
import { ProfilePage } from './pages/ProfilePage'
import { RegisterPage } from './pages/RegisterPage'
import { RootRedirect } from './pages/RootRedirect'
import { WorkspacePage } from './pages/WorkspacePage'

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />

      {/* An invitation link works logged out too: it shows a preview first */}
      <Route path="/invite/:token" element={<AcceptInvitePage />} />

      {/* Everything inside needs a login, and shares the header */}
      <Route element={<ProtectedRoute />}>
        <Route element={<AppLayout />}>
          <Route path="/" element={<RootRedirect />} />
          <Route path="/workspaces/new" element={<CreateWorkspacePage />} />
          <Route path="/w/:slug" element={<WorkspacePage />} />
          <Route path="/w/:slug/members" element={<MembersPage />} />
          <Route path="/profile" element={<ProfilePage />} />
        </Route>
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default App