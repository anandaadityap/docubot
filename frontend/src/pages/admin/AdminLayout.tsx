import { useEffect, useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { setToken } from '../../api/client'
import { cn } from '../../lib/format'
import { Button } from '../../components/ui/Button'

const links = [
  { to: '/admin/documents', label: 'Dokumen' },
  { to: '/admin/install', label: 'Pasang' },
  { to: '/admin/conversations', label: 'Percakapan' },
  { to: '/admin/analytics', label: 'Analitik' },
  { to: '/admin/settings', label: 'Setelan' },
]

export function AdminLayout() {
  const nav = useNavigate()
  const [open, setOpen] = useState(false)

  useEffect(() => {
    function onUnauthorized() {
      setToken(null)
      nav('/login?reason=expired', { replace: true })
    }
    window.addEventListener('docubot:unauthorized', onUnauthorized)
    return () => window.removeEventListener('docubot:unauthorized', onUnauthorized)
  }, [nav])

  function logout() {
    setToken(null)
    nav('/login', { replace: true })
  }

  const navItems = (
    <>
      {links.map((l) => (
        <NavLink
          key={l.to}
          to={l.to}
          onClick={() => setOpen(false)}
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
    </>
  )

  return (
    <div className="flex h-full min-h-full flex-col md:flex-row">
      <header className="flex items-center justify-between border-b border-slate-200 bg-white px-4 py-3 md:hidden">
        <div className="flex items-center gap-2">
          <span className="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-brand text-sm font-semibold text-white">D</span>
          <span className="font-semibold text-ink">DocuBot</span>
        </div>
        <button type="button" className="rounded-lg px-3 py-2 text-sm text-slate-600 hover:bg-slate-50" onClick={() => setOpen((v) => !v)}>
          Menu
        </button>
      </header>
      {open && (
        <div className="fixed inset-0 z-30 bg-slate-900/40 md:hidden" onClick={() => setOpen(false)} role="presentation">
          <aside className="flex h-full w-64 flex-col bg-white p-3" onClick={(e) => e.stopPropagation()}>
            <nav className="flex flex-1 flex-col gap-1">{navItems}</nav>
            <Button variant="ghost" className="w-full justify-start" onClick={logout}>
              Keluar
            </Button>
          </aside>
        </div>
      )}
      <aside className="hidden w-56 shrink-0 flex-col border-r border-slate-200 bg-white md:flex">
        <div className="flex items-center gap-2 px-4 py-4">
          <span className="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-brand text-sm font-semibold text-white">D</span>
          <span className="font-semibold text-ink">DocuBot</span>
        </div>
        <nav className="flex flex-1 flex-col gap-1 px-2">{navItems}</nav>
        <div className="p-3">
          <Button variant="ghost" className="w-full justify-start" onClick={logout}>
            Keluar
          </Button>
        </div>
      </aside>
      <main className="flex-1 overflow-auto p-4 md:p-6">
        <Outlet />
      </main>
    </div>
  )
}
