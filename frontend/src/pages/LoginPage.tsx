import { type FormEvent, type ReactNode, useEffect, useState } from 'react'
import { Link, Navigate, useNavigate, useSearchParams } from 'react-router-dom'
import { authApi } from '../api/admin'
import { authPublicApi } from '../api/chat'
import { getToken, HttpError, setToken } from '../api/client'
import { Button } from '../components/ui/Button'
import { Input, Label } from '../components/ui/Input'

export function LoginPage() {
  const nav = useNavigate()
  const [params] = useSearchParams()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState(params.get('reason') === 'expired' ? 'Sesi berakhir. Silakan login lagi.' : '')
  const [busy, setBusy] = useState(false)
  const [registerOpen, setRegisterOpen] = useState(false)

  useEffect(() => {
    authPublicApi
      .registerStatus()
      .then((s) => setRegisterOpen(s.open))
      .catch(() => setRegisterOpen(false))
  }, [])

  if (getToken()) {
    return <Navigate to="/admin/install" replace />
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      const res = await authApi.login(email, password)
      setToken(res.token)
      nav('/admin/install', { replace: true })
    } catch (err) {
      setError(err instanceof HttpError ? err.message : 'login gagal')
    } finally {
      setBusy(false)
    }
  }

  return (
    <AuthShell title="Masuk admin" subtitle="Kelola dokumen, percakapan, dan pengaturan bot.">
      <form className="space-y-4" onSubmit={onSubmit}>
        <div>
          <Label>Email</Label>
          <Input type="email" autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
        </div>
        <div>
          <Label>Password</Label>
          <Input type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} required />
        </div>
        {error && <p className="text-sm text-red-600">{error}</p>}
        <Button type="submit" className="w-full" disabled={busy}>
          {busy ? 'Masuk...' : 'Masuk'}
        </Button>
      </form>
      {registerOpen && (
        <p className="mt-4 text-center text-sm text-muted">
          Belum punya akun?{' '}
          <Link to="/register" className="text-brand hover:underline">
            Daftar
          </Link>
        </p>
      )}
    </AuthShell>
  )
}

export function RegisterPage() {
  const nav = useNavigate()
  const [params] = useSearchParams()
  const invite = params.get('invite') || ''
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [open, setOpen] = useState<boolean | null>(null)

  useEffect(() => {
    authPublicApi
      .registerStatus()
      .then((s) => setOpen(s.open || Boolean(invite)))
      .catch(() => setOpen(Boolean(invite)))
  }, [invite])

  if (getToken()) {
    return <Navigate to="/admin/install" replace />
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      await authApi.register(name, email, password, invite || undefined)
      const res = await authApi.login(email, password)
      setToken(res.token)
      nav('/admin/install', { replace: true })
    } catch (err) {
      setError(err instanceof HttpError ? err.message : 'registrasi gagal')
    } finally {
      setBusy(false)
    }
  }

  if (open === false) {
    return (
      <AuthShell title="Pendaftaran ditutup" subtitle="Akun admin sudah ada. Demo live tidak menerima daftar publik.">
        <p className="text-sm text-muted">
          Masuk dengan akun yang sudah terdaftar, atau minta tautan undangan dari pemilik bot.
        </p>
        <Link to="/login" className="mt-4 inline-block text-sm text-brand hover:underline">
          Ke halaman masuk
        </Link>
      </AuthShell>
    )
  }

  return (
    <AuthShell title="Daftar admin" subtitle="Satu akun pertama menjadi owner bot publik. Pendaftaran ditutup setelah itu.">
      {open === null ? (
        <p className="text-sm text-muted">Memuat...</p>
      ) : (
        <>
          <form className="space-y-4" onSubmit={onSubmit}>
            <div>
              <Label>Nama</Label>
              <Input value={name} onChange={(e) => setName(e.target.value)} required />
            </div>
            <div>
              <Label>Email</Label>
              <Input type="email" autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
            </div>
            <div>
              <Label>Password (min. 8 karakter)</Label>
              <Input type="password" autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} minLength={8} required />
            </div>
            {error && <p className="text-sm text-red-600">{error}</p>}
            <Button type="submit" className="w-full" disabled={busy}>
              {busy ? 'Mendaftar...' : 'Daftar'}
            </Button>
          </form>
          <p className="mt-4 text-center text-sm text-muted">
            Sudah punya akun?{' '}
            <Link to="/login" className="text-brand hover:underline">
              Masuk
            </Link>
          </p>
        </>
      )}
    </AuthShell>
  )
}

function AuthShell({ title, subtitle, children }: { title: string; subtitle: string; children: ReactNode }) {
  return (
    <div className="flex min-h-full items-center justify-center px-4">
      <div className="w-full max-w-md rounded-2xl border border-slate-200 bg-white p-8 shadow-sm">
        <Link to="/" className="mb-6 inline-flex items-center gap-2">
          <span className="inline-flex h-9 w-9 items-center justify-center rounded-xl bg-brand text-sm font-semibold text-white">D</span>
          <span className="font-semibold text-ink">DocuBot</span>
        </Link>
        <h1 className="text-xl font-semibold text-ink">{title}</h1>
        <p className="mt-1 mb-6 text-sm text-muted">{subtitle}</p>
        {children}
      </div>
    </div>
  )
}
