import { useCallback, useEffect, useRef, useState } from 'react'
import { documentsApi } from '../../api/admin'
import { HttpError } from '../../api/client'
import type { Document } from '../../api/types'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { formatBytes, formatDate } from '../../lib/format'

type ChunkPreview = { position: number; content: string; token_count: number }

export function DocumentsPage() {
  const [docs, setDocs] = useState<Document[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [previewId, setPreviewId] = useState<number | null>(null)
  const [chunks, setChunks] = useState<ChunkPreview[]>([])
  const inputRef = useRef<HTMLInputElement>(null)

  const load = useCallback(async () => {
    try {
      setDocs(await documentsApi.list())
    } catch (e) {
      setError(e instanceof HttpError ? e.message : 'gagal memuat dokumen')
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    const pending = docs.some((d) => d.status === 'pending' || d.status === 'processing')
    if (!pending) return
    const t = window.setInterval(() => void load(), 1500)
    return () => window.clearInterval(t)
  }, [docs, load])

  async function onFiles(files: FileList | null) {
    if (!files?.length) return
    setBusy(true)
    setError('')
    try {
      for (const file of Array.from(files)) {
        await documentsApi.upload(file)
      }
      await load()
    } catch (e) {
      setError(e instanceof HttpError ? e.message : 'upload gagal')
    } finally {
      setBusy(false)
      if (inputRef.current) inputRef.current.value = ''
    }
  }

  async function remove(id: number) {
    if (!window.confirm('Hapus dokumen ini beserta chunk-nya?')) return
    try {
      await documentsApi.remove(id)
      if (previewId === id) {
        setPreviewId(null)
        setChunks([])
      }
      await load()
    } catch (e) {
      setError(e instanceof HttpError ? e.message : 'gagal menghapus')
    }
  }

  async function reprocess(id: number) {
    try {
      await documentsApi.reprocess(id)
      await load()
    } catch (e) {
      setError(e instanceof HttpError ? e.message : 'gagal memproses ulang')
    }
  }

  async function preview(id: number) {
    if (previewId === id) {
      setPreviewId(null)
      setChunks([])
      return
    }
    try {
      const d = await documentsApi.get(id)
      setPreviewId(id)
      setChunks(d.chunks ?? [])
    } catch (e) {
      setError(e instanceof HttpError ? e.message : 'gagal memuat cuplikan')
    }
  }

  const empty = docs.length === 0

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-ink">Dokumen</h1>
          <p className="text-sm text-muted">Unggah `.md` atau `.txt` (maks 5 MB) untuk knowledge base.</p>
        </div>
        <div>
          <input
            ref={inputRef}
            type="file"
            accept=".md,.txt,text/markdown,text/plain"
            className="hidden"
            multiple
            onChange={(e) => void onFiles(e.target.files)}
          />
          <Button disabled={busy} onClick={() => inputRef.current?.click()}>
            {busy ? 'Mengunggah...' : 'Upload .md / .txt'}
          </Button>
        </div>
      </div>
      {error && <p className="mb-3 text-sm text-red-600">{error}</p>}

      {empty && (
        <div className="mb-6 rounded-xl border border-dashed border-slate-300 bg-white p-6">
          <h2 className="text-sm font-semibold text-ink">Mulai dalam 3 langkah</h2>
          <ol className="mt-3 list-decimal space-y-2 pl-5 text-sm text-muted">
            <li>Unggah file FAQ atau panduan (`.md` / `.txt`).</li>
            <li>Tunggu status menjadi <span className="font-medium text-ink">Ready</span>.</li>
            <li>
              Buka{' '}
              <a href="/" className="text-brand hover:underline" target="_blank" rel="noreferrer">
                halaman chat publik
              </a>{' '}
              dan coba tanya.
            </li>
          </ol>
          <p className="mt-4 text-sm text-muted">
            Belum punya file?{' '}
            <a href="/samples/faq-contoh.md" download className="text-brand hover:underline">
              Unduh contoh FAQ
            </a>{' '}
            lalu unggah di sini.
          </p>
        </div>
      )}

      <div className="hidden overflow-hidden rounded-xl border border-slate-200 bg-white md:block">
        <table className="w-full text-left text-sm">
          <thead className="bg-slate-50 text-xs uppercase tracking-wide text-muted">
            <tr>
              <th className="px-4 py-3 font-medium">Filename</th>
              <th className="px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3 font-medium">Chunks</th>
              <th className="px-4 py-3 font-medium">Ukuran</th>
              <th className="px-4 py-3 font-medium">Diunggah</th>
              <th className="px-4 py-3 font-medium">Aksi</th>
            </tr>
          </thead>
          <tbody>
            {docs.map((d) => (
              <DocTableRow
                key={d.id}
                d={d}
                previewId={previewId}
                onPreview={() => void preview(d.id)}
                onReprocess={() => void reprocess(d.id)}
                onRemove={() => void remove(d.id)}
              />
            ))}
          </tbody>
        </table>
      </div>

      <div className="space-y-3 md:hidden">
        {docs.map((d) => (
          <div key={d.id} className="rounded-xl border border-slate-200 bg-white p-4">
            <p className="font-medium text-ink">{d.filename}</p>
            <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-muted">
              <Badge status={d.status} />
              <span>{d.status === 'ready' ? `${d.chunk_count} chunk` : '—'}</span>
              <span>{formatBytes(d.size_bytes)}</span>
            </div>
            {d.status === 'failed' && d.error_msg && <p className="mt-2 text-xs text-red-600">{d.error_msg}</p>}
            <div className="mt-3 flex flex-wrap gap-1">
              <Button variant="outline" onClick={() => void preview(d.id)}>
                {previewId === d.id ? 'Tutup cuplikan' : 'Cuplikan'}
              </Button>
              <Button variant="outline" onClick={() => void reprocess(d.id)}>
                Proses ulang
              </Button>
              <Button variant="danger" onClick={() => void remove(d.id)}>
                Hapus
              </Button>
            </div>
          </div>
        ))}
      </div>

      {previewId != null && (
        <div className="mt-4 rounded-xl border border-slate-200 bg-white p-4">
          <h2 className="mb-3 text-sm font-semibold text-ink">Cuplikan chunk</h2>
          {chunks.length === 0 ? (
            <p className="text-sm text-muted">Belum ada chunk. Tunggu status Ready atau proses ulang.</p>
          ) : (
            <ul className="space-y-3">
              {chunks.map((c) => (
                <li key={c.position} className="rounded-lg bg-slate-50 p-3 text-sm">
                  <p className="text-[11px] uppercase tracking-wide text-muted">
                    #{c.position} · {c.token_count} token
                  </p>
                  <p className="mt-1 whitespace-pre-wrap text-ink">{c.content}</p>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}

function DocTableRow({
  d,
  previewId,
  onPreview,
  onReprocess,
  onRemove,
}: {
  d: Document
  previewId: number | null
  onPreview: () => void
  onReprocess: () => void
  onRemove: () => void
}) {
  return (
    <tr className="border-t border-slate-100">
      <td className="px-4 py-3 font-medium text-ink">
        {d.filename}
        {d.embed_model && <p className="text-[11px] font-normal text-muted">{d.embed_model}</p>}
      </td>
      <td className="px-4 py-3">
        <Badge status={d.status} />
        {d.status === 'failed' && d.error_msg && <p className="mt-1 max-w-xs text-xs text-red-600">{d.error_msg}</p>}
      </td>
      <td className="px-4 py-3 text-muted">{d.status === 'ready' ? d.chunk_count : '—'}</td>
      <td className="px-4 py-3 text-muted">{formatBytes(d.size_bytes)}</td>
      <td className="px-4 py-3 text-muted">{formatDate(d.created_at)}</td>
      <td className="px-4 py-3">
        <div className="flex flex-wrap gap-1">
          <Button variant="outline" onClick={onPreview}>
            {previewId === d.id ? 'Tutup' : 'Cuplikan'}
          </Button>
          <Button variant="outline" onClick={onReprocess}>
            Proses ulang
          </Button>
          <Button variant="danger" onClick={onRemove}>
            Hapus
          </Button>
        </div>
      </td>
    </tr>
  )
}
