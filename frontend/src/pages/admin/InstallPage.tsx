import { type FormEvent, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { adminBotApi } from '../../api/admin'
import { HttpError } from '../../api/client'
import type { AdminBot } from '../../api/types'
import { ChatWindow } from '../../components/chat/ChatWindow'
import { Button } from '../../components/ui/Button'
import { Input, Label, Textarea } from '../../components/ui/Input'

function iframeSnippet(origin: string, slug: string) {
  return `<iframe
  src="${origin}/b/${slug}?embed=1"
  title="DocuBot"
  style="width:100%;height:640px;border:0;border-radius:12px"
  loading="lazy"
></iframe>`
}

async function copyText(value: string) {
  await navigator.clipboard.writeText(value)
}

export function InstallPage() {
  const [bot, setBot] = useState<AdminBot | null>(null)
  const [slug, setSlug] = useState('')
  const [name, setName] = useState('')
  const [welcome, setWelcome] = useState('')
  const [active, setActive] = useState(true)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState('')
  const [copied, setCopied] = useState('')
  const [busy, setBusy] = useState(false)
  const origin = typeof window !== 'undefined' ? window.location.origin : ''

  useEffect(() => {
    adminBotApi
      .get()
      .then((b) => {
        setBot(b)
        setSlug(b.slug)
        setName(b.name)
        setWelcome(b.welcome_message)
        setActive(b.active)
      })
      .catch((e) => setError(e instanceof HttpError ? e.message : 'gagal memuat bot'))
  }, [])

  async function onSave(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    setSaved('')
    try {
      const next = await adminBotApi.put({ slug, name, welcome_message: welcome, active })
      setBot(next)
      setSlug(next.slug)
      setName(next.name)
      setWelcome(next.welcome_message)
      setActive(next.active)
      setSaved('Tersimpan.')
    } catch (err) {
      setError(err instanceof HttpError ? err.message : 'gagal menyimpan')
    } finally {
      setBusy(false)
    }
  }

  async function copy(label: string, value: string) {
    try {
      await copyText(value)
      setCopied(label)
      window.setTimeout(() => setCopied(''), 1500)
    } catch {
      setError('Gagal menyalin. Salin manual dari kotak teks.')
    }
  }

  const publicURL = bot ? `${origin}${bot.public_path}` : ''
  const snippet = bot ? iframeSnippet(origin, bot.slug) : ''

  return (
    <div className="mx-auto max-w-3xl">
      <h1 className="text-xl font-semibold text-ink">Pasang</h1>
      <p className="mb-4 text-sm text-muted">1) unggah dokumen 2) tes di kotak di bawah 3) salin snippet iframe.</p>

      <div className="mb-6 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
        Ganti slug = putus embed lama. Perbarui iframe di situs klien setelah menyimpan slug baru.
      </div>

      <form className="space-y-4 rounded-xl border border-slate-200 bg-white p-6" onSubmit={(e) => void onSave(e)}>
        <div>
          <Label>Slug publik</Label>
          <Input value={slug} onChange={(e) => setSlug(e.target.value)} required />
          <p className="mt-1 text-xs text-muted">Huruf kecil, angka, dan tanda hubung. 3–48 karakter.</p>
        </div>
        <div>
          <Label>Nama bot</Label>
          <Input value={name} onChange={(e) => setName(e.target.value)} required />
        </div>
        <div>
          <Label>Welcome message</Label>
          <Textarea rows={3} value={welcome} onChange={(e) => setWelcome(e.target.value)} required />
        </div>
        <label className="flex items-center gap-2 text-sm text-ink">
          <input type="checkbox" checked={active} onChange={(e) => setActive(e.target.checked)} />
          Bot aktif
        </label>
        {error && <p className="text-sm text-red-600">{error}</p>}
        {saved && <p className="text-sm text-emerald-600">{saved}</p>}
        <Button type="submit" disabled={busy || !bot}>
          {busy ? 'Menyimpan...' : 'Simpan'}
        </Button>
      </form>

      {bot && (
        <div className="mt-6 space-y-4 rounded-xl border border-slate-200 bg-white p-6">
          <div>
            <Label>URL publik</Label>
            <div className="mt-1 flex gap-2">
              <Input readOnly value={publicURL} />
              <Button type="button" variant="outline" onClick={() => void copy('url', publicURL)}>
                {copied === 'url' ? 'Tersalin' : 'Salin'}
              </Button>
            </div>
            <a href={bot.public_path} target="_blank" rel="noreferrer" className="mt-2 inline-block text-sm text-brand hover:underline">
              Buka chat publik
            </a>
          </div>
          <div>
            <Label>Snippet iframe</Label>
            <textarea
              readOnly
              className="mt-1 w-full rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 font-mono text-xs"
              rows={7}
              value={snippet}
            />
            <Button type="button" variant="outline" className="mt-2" onClick={() => void copy('iframe', snippet)}>
              {copied === 'iframe' ? 'Tersalin' : 'Salin snippet'}
            </Button>
          </div>
        </div>
      )}

      {bot && (
        <div className="mt-6">
          <h2 className="mb-2 text-sm font-semibold text-ink">Tes (playground)</h2>
          <p className="mb-3 text-xs text-muted">Percakapan ini tidak muncul di daftar Percakapan pelanggan.</p>
          <div className="h-[480px] overflow-hidden rounded-xl border border-slate-200 bg-white">
            <ChatWindow key={bot.slug} slug={bot.slug} variant="embed" channel="playground" />
          </div>
        </div>
      )}

      <p className="mt-6 text-sm text-muted">
        Belum unggah dokumen?{' '}
        <Link to="/admin/documents" className="text-brand hover:underline">
          Ke halaman Dokumen
        </Link>
        .
      </p>
    </div>
  )
}
