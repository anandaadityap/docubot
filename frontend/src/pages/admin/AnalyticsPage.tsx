import { useEffect, useState } from 'react'
import { CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { analyticsApi } from '../../api/admin'
import { HttpError } from '../../api/client'
import type { Overview, TopQuestion } from '../../api/types'
import { formatDay } from '../../lib/format'

export function AnalyticsPage() {
  const [ov, setOv] = useState<Overview | null>(null)
  const [top, setTop] = useState<TopQuestion[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    Promise.all([analyticsApi.overview(), analyticsApi.topQuestions(10)])
      .then(([a, b]) => {
        setOv(a)
        setTop(b)
      })
      .catch((e) => setError(e instanceof HttpError ? e.message : 'gagal memuat analitik'))
  }, [])

  const chart = (ov?.daily ?? []).map((d) => ({ ...d, label: formatDay(d.date) }))

  return (
    <div>
      <h1 className="text-xl font-semibold text-ink">Analitik</h1>
      <p className="mb-6 text-sm text-muted">Volume chat 14 hari terakhir, latensi, dan pertanyaan teratas.</p>
      {error && <p className="mb-3 text-sm text-red-600">{error}</p>}
      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Stat label="Total percakapan" value={ov?.total_conversations ?? '—'} />
        <Stat label="Total pesan" value={ov?.total_messages ?? '—'} />
        <Stat label="Rata-rata latensi" value={ov ? `${ov.avg_latency_ms} ms` : '—'} />
      </div>
      <div className="mb-6 rounded-xl border border-slate-200 bg-white p-4">
        <h2 className="mb-4 text-sm font-semibold text-ink">Chat per hari</h2>
        <div className="h-64">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chart}>
              <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
              <XAxis dataKey="label" tick={{ fontSize: 12 }} />
              <YAxis allowDecimals={false} tick={{ fontSize: 12 }} />
              <Tooltip />
              <Line type="monotone" dataKey="chats" stroke="#4f46e5" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>
      <div className="rounded-xl border border-slate-200 bg-white">
        <h2 className="border-b border-slate-100 px-4 py-3 text-sm font-semibold text-ink">Top 10 pertanyaan</h2>
        <table className="w-full text-left text-sm">
          <thead className="text-xs uppercase tracking-wide text-muted">
            <tr>
              <th className="px-4 py-2 font-medium">Pertanyaan</th>
              <th className="px-4 py-2 font-medium">Jumlah</th>
            </tr>
          </thead>
          <tbody>
            {top.length === 0 && (
              <tr>
                <td colSpan={2} className="px-4 py-8 text-center text-muted">
                  Belum ada data.
                </td>
              </tr>
            )}
            {top.map((q) => (
              <tr key={q.question} className="border-t border-slate-100">
                <td className="px-4 py-2 text-ink">{q.question}</td>
                <td className="px-4 py-2 text-muted">{q.count}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-xl border border-slate-200 bg-white p-4">
      <p className="text-xs font-medium uppercase tracking-wide text-muted">{label}</p>
      <p className="mt-1 text-2xl font-semibold text-ink">{value}</p>
    </div>
  )
}
