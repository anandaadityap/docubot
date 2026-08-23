import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { authPublicApi, botApi } from '../api/chat'
import { Button } from '../components/ui/Button'

const iframeExample = `<iframe
  src="https://your-host/b/your-bot?embed=1"
  title="DocuBot"
  style="width:100%;height:640px;border:0;border-radius:12px"
  loading="lazy"
></iframe>`

export function LandingPage() {
  const [demo, setDemo] = useState<{ slug?: string; configured: boolean } | null>(null)
  const [registerOpen, setRegisterOpen] = useState(false)

  useEffect(() => {
    botApi
      .demo()
      .then((d) => setDemo({ slug: d.slug, configured: d.configured }))
      .catch(() => setDemo({ configured: false }))
    authPublicApi
      .registerStatus()
      .then((s) => setRegisterOpen(s.open))
      .catch(() => setRegisterOpen(false))
  }, [])

  const demoReady = Boolean(demo?.configured && demo.slug)
  const snippet = iframeExample.replace('https://your-host', typeof window !== 'undefined' ? window.location.origin : 'https://your-host')

  return (
    <div className="min-h-full bg-surface">
      <header className="mx-auto flex max-w-3xl items-center justify-between px-4 py-5">
        <div className="flex items-center gap-2">
          <span className="inline-flex h-9 w-9 items-center justify-center rounded-xl bg-brand text-sm font-semibold text-white">D</span>
          <span className="font-semibold text-ink">DocuBot</span>
        </div>
        <div className="flex items-center gap-3">
          {registerOpen && (
            <Link to="/register" className="text-sm text-muted hover:text-brand">
              Daftar
            </Link>
          )}
          <Link to="/login" className="text-sm font-medium text-brand hover:underline">
            Masuk admin
          </Link>
        </div>
      </header>

      <main className="mx-auto max-w-3xl px-4 pb-16">
        <h1 className="text-3xl font-semibold tracking-tight text-ink">Support bot dari dokumen Anda</h1>
        <p className="mt-3 max-w-xl text-sm leading-6 text-muted">
          Aplikasi self-host: unggah FAQ atau panduan (Markdown/TXT), tes jawabannya, lalu tempel iframe ke situs klien. Satu
          akun, satu bot, satu slug publik.
        </p>

        <div className="mt-6 flex flex-wrap gap-3">
          {demoReady ? (
            <Link to={`/b/${demo!.slug}`}>
              <Button>Coba demo</Button>
            </Link>
          ) : (
            <Button disabled>{demo === null ? 'Memuat demo...' : 'Demo belum tersedia'}</Button>
          )}
          {registerOpen ? (
            <Link to="/register">
              <Button variant="outline">Daftar</Button>
            </Link>
          ) : (
            <Link to="/login">
              <Button variant="outline">Masuk admin</Button>
            </Link>
          )}
        </div>
        {demo && !demo.configured && (
          <p className="mt-2 text-xs text-muted">Belum ada bot. Daftar sebagai admin pertama, unggah dokumen, lalu coba lagi.</p>
        )}

        <ol className="mt-10 grid gap-3 sm:grid-cols-3">
          {[
            { n: '1', t: 'Unggah dokumen', d: 'Markdown atau TXT jadi knowledge base.' },
            { n: '2', t: 'Tes chat', d: 'Jawaban streaming plus kutipan sumber.' },
            { n: '3', t: 'Salin iframe', d: 'Tempel ke situs klien — API tetap di DocuBot.' },
          ].map((s) => (
            <li key={s.n} className="rounded-2xl border border-slate-200 bg-white p-4">
              <p className="text-xs font-semibold text-brand">{s.n}</p>
              <p className="mt-1 text-sm font-semibold text-ink">{s.t}</p>
              <p className="mt-1 text-xs leading-5 text-muted">{s.d}</p>
            </li>
          ))}
        </ol>

        <figure className="mt-6 rounded-2xl border border-slate-200 bg-white p-5">
          <figcaption className="text-sm font-semibold text-ink">Contoh jawaban berkutipan</figcaption>
          <p className="mt-3 text-sm text-muted">Pengunjung: Berapa lama pengembalian dana?</p>
          <p className="mt-2 text-sm leading-6 text-ink">
            Pengembalian dana diproses 3–7 hari kerja setelah barang diterima.{' '}
            <span className="rounded-md bg-brand/10 px-1.5 py-0.5 text-xs font-medium text-brand">[1]</span>
          </p>
          <p className="mt-3 text-xs text-muted">Sumber [1] · kebijakan-refund.md</p>
        </figure>

        <section className="mt-6 rounded-2xl border border-slate-200 bg-white p-5">
          <h2 className="text-sm font-semibold text-ink">Pasang ke situs lain</h2>
          <p className="mt-1 text-sm text-muted">
            Tempel iframe ke halaman HTML klien. Ganti <code>your-bot</code> dengan slug Anda.
          </p>
          <pre className="mt-3 overflow-x-auto rounded-xl bg-slate-50 p-4 text-xs text-ink">{snippet}</pre>
        </section>

        <p className="mt-8 text-xs text-muted">Go · React · SQLite · RAG</p>
      </main>
    </div>
  )
}
