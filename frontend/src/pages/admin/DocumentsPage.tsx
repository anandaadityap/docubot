import { useCallback, useEffect, useRef, useState } from 'react'
import { documentsApi } from '../../api/admin'
import { HttpError } from '../../api/client'
import type { Document } from '../../api/types'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { formatBytes, formatDate } from '../../lib/format'

export function DocumentsPage() {
  const [docs, setDocs] = useState<Document[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
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
      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white">
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
            {docs.length === 0 && (
              <tr>
                <td colSpan={6} className="px-4 py-10 text-center text-muted">
                  Belum ada dokumen. Upload file sample di `backend/testdata/manual-pengguna.md`.
                </td>
              </tr>
            )}
            {docs.map((d) => (
              <tr key={d.id} className="border-t border-slate-100">
                <td className="px-4 py-3 font-medium text-ink">{d.filename}</td>
                <td className="px-4 py-3">
                  <Badge status={d.status} />
                  {d.status === 'failed' && d.error_msg && (
                    <p className="mt-1 max-w-xs text-xs text-red-600">{d.error_msg}</p>
                  )}
                </td>
                <td className="px-4 py-3 text-muted">{d.status === 'ready' ? d.chunk_count : '—'}</td>
                <td className="px-4 py-3 text-muted">{formatBytes(d.size_bytes)}</td>
                <td className="px-4 py-3 text-muted">{formatDate(d.created_at)}</td>
                <td className="px-4 py-3">
                  <div className="flex gap-1">
                    <Button variant="outline" onClick={() => void reprocess(d.id)}>
                      Proses ulang
                    </Button>
                    <Button variant="danger" onClick={() => void remove(d.id)}>
                      Hapus
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
