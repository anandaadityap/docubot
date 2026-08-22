import { type FormEvent, type ReactNode, useState } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { authApi } from '../api/admin'
import { getToken, HttpError, setToken } from '../api/client'
import { Button } from '../components/ui/Button'
import { Input, Label } from '../components/ui/Input'

export function LoginPage() {
  const nav = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  if (getToken()) {
    return <Navigate to="/admin/documents" replace />
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      const res = await authApi.login(email, password)
      setToken(res.token)
      nav('/admin/documents', { replace: true })
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
      <p className="mt-4 text-center text-sm text-muted">
        Belum punya akun?{' '}
        <Link to="/register" className="text-brand hover:underline">
          Daftar
        </Link>
      </p>
    </AuthShell>
  )
}

export function RegisterPage() {
  const nav = useNavigate()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  if (getToken()) {
    return <Navigate to="/admin/documents" replace />
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      await authApi.register(name, email, password)
      const res = await authApi.login(email, password)
      setToken(res.token)
      nav('/admin/documents', { replace: true })
    } catch (err) {
      setError(err instanceof HttpError ? err.message : 'registrasi gagal')
    } finally {
      setBusy(false)
    }
  }

  return (
    <AuthShell title="Daftar admin" subtitle="Satu akun untuk satu bot knowledge base.">
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
