import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { setToken } from '../../api/client'
import { cn } from '../../lib/format'
import { Button } from '../../components/ui/Button'

const links = [
  { to: '/admin/documents', label: 'Dokumen' },
  { to: '/admin/conversations', label: 'Percakapan' },
  { to: '/admin/analytics', label: 'Analitik' },
  { to: '/admin/settings', label: 'Setelan' },
]

export function AdminLayout() {
  const nav = useNavigate()
  function logout() {
    setToken(null)
    nav('/login', { replace: true })
  }
  return (
    <div className="flex h-full min-h-full">
      <aside className="flex w-56 shrink-0 flex-col border-r border-slate-200 bg-white">
        <div className="flex items-center gap-2 px-4 py-4">
          <span className="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-brand text-sm font-semibold text-white">D</span>
          <span className="font-semibold text-ink">DocuBot</span>
        </div>
        <nav className="flex flex-1 flex-col gap-1 px-2">
          {links.map((l) => (
            <NavLink
              key={l.to}
              to={l.to}
              className={({ isActive }) =>
                cn(
                  'rounded-lg px-3 py-2 text-sm font-medium',
                  isActive ? 'bg-brand/10 text-brand' : 'text-slate-600 hover:bg-slate-50',
                )
              }
            >
              {l.label}
            </NavLink>
          ))}
        </nav>
        <div className="p-3">
          <Button variant="ghost" className="w-full justify-start" onClick={logout}>
            Keluar
          </Button>
        </div>
      </aside>
      <main className="flex-1 overflow-auto p-6">
        <Outlet />
      </main>
    </div>
  )
}
