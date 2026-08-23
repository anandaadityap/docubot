import { BrowserRouter, Navigate, Outlet, Route, Routes } from 'react-router-dom'
import { getToken } from './api/client'
import { LandingPage } from './pages/LandingPage'
import { PublicChatPage } from './pages/PublicChatPage'
import { LoginPage, RegisterPage } from './pages/LoginPage'
import { AdminLayout } from './pages/admin/AdminLayout'
import { DocumentsPage } from './pages/admin/DocumentsPage'
import { ConversationDetailPage, ConversationsPage } from './pages/admin/ConversationsPage'
import { AnalyticsPage } from './pages/admin/AnalyticsPage'
import { SettingsPage } from './pages/admin/SettingsPage'
import { InstallPage } from './pages/admin/InstallPage'

function RequireAuth() {
  if (!getToken()) {
    return <Navigate to="/login" replace />
  }
  return <Outlet />
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<LandingPage />} />
        <Route path="/b/:slug" element={<PublicChatPage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route element={<RequireAuth />}>
          <Route path="/admin" element={<AdminLayout />}>
            <Route index element={<Navigate to="documents" replace />} />
            <Route path="documents" element={<DocumentsPage />} />
            <Route path="install" element={<InstallPage />} />
            <Route path="conversations" element={<ConversationsPage />} />
            <Route path="conversations/:id" element={<ConversationDetailPage />} />
            <Route path="analytics" element={<AnalyticsPage />} />
            <Route path="settings" element={<SettingsPage />} />
          </Route>
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
