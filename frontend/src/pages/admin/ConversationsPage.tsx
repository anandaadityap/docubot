import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { conversationsApi } from '../../api/admin'
import { HttpError } from '../../api/client'
import type { ChatMessage, Conversation } from '../../api/types'
import { MessageBubble } from '../../components/chat/MessageBubble'
import { Button } from '../../components/ui/Button'
import { formatDate } from '../../lib/format'

export function ConversationsPage() {
  const [items, setItems] = useState<Conversation[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [error, setError] = useState('')
  const limit = 20

  useEffect(() => {
    conversationsApi
      .list(page, limit)
      .then((res) => {
        setItems(res.items)
        setTotal(res.total)
      })
      .catch((e) => setError(e instanceof HttpError ? e.message : 'gagal memuat'))
  }, [page])

  const pages = Math.max(1, Math.ceil(total / limit))

  return (
    <div>
      <h1 className="text-xl font-semibold text-ink">Percakapan</h1>
      <p className="mb-6 text-sm text-muted">Log chat pengunjung, {total} sesi.</p>
      {error && <p className="mb-3 text-sm text-red-600">{error}</p>}
      <div className="overflow-x-auto rounded-xl border border-slate-200 bg-white">
        <table className="w-full text-left text-sm">
          <thead className="bg-slate-50 text-xs uppercase tracking-wide text-muted">
            <tr>
              <th className="px-4 py-3 font-medium">Judul</th>
              <th className="px-4 py-3 font-medium">Pesan</th>
              <th className="px-4 py-3 font-medium">Waktu</th>
            </tr>
          </thead>
          <tbody>
            {items.length === 0 && (
              <tr>
                <td colSpan={3} className="px-4 py-10 text-center text-muted">
                  Belum ada percakapan. Coba chat di halaman publik.
                </td>
              </tr>
            )}
            {items.map((c) => (
              <tr key={c.id} className="border-t border-slate-100 hover:bg-slate-50">
                <td className="px-4 py-3">
                  <Link to={`/admin/conversations/${c.id}`} className="font-medium text-ink hover:text-brand">
                    {c.title}
                  </Link>
                </td>
                <td className="px-4 py-3 text-muted">{c.message_count}</td>
                <td className="px-4 py-3 text-muted">{formatDate(c.updated_at || c.created_at)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {pages > 1 && (
        <div className="mt-4 flex items-center gap-2">
          <Button variant="outline" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
            Sebelumnya
          </Button>
          <span className="text-sm text-muted">
            {page} / {pages}
          </span>
          <Button variant="outline" disabled={page >= pages} onClick={() => setPage((p) => p + 1)}>
            Berikutnya
          </Button>
        </div>
      )}
    </div>
  )
}

export function ConversationDetailPage() {
  const { id } = useParams()
  const [title, setTitle] = useState('')
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    const n = Number(id)
    if (!n) return
    conversationsApi
      .get(n)
      .then((d) => {
        setTitle(d.title)
        setMessages(d.messages)
      })
      .catch((e) => setError(e instanceof HttpError ? e.message : 'gagal memuat'))
  }, [id])

  return (
    <div>
      <Link to="/admin/conversations" className="text-sm text-brand hover:underline">
        ← Semua percakapan
      </Link>
      <h1 className="mt-2 text-xl font-semibold text-ink">{title || 'Percakapan'}</h1>
      {error && <p className="mt-2 text-sm text-red-600">{error}</p>}
      <div className="mt-6 flex max-w-2xl flex-col gap-3">
        {messages.map((m) => (
          <MessageBubble key={m.id} role={m.role} content={m.content} sources={m.sources} />
        ))}
      </div>
    </div>
  )
}
