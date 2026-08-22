import { type FormEvent, useEffect, useState } from 'react'
import { settingsApi } from '../../api/admin'
import { HttpError } from '../../api/client'
import type { Settings } from '../../api/types'
import { Button } from '../../components/ui/Button'
import { Input, Label, Textarea } from '../../components/ui/Input'

const empty: Settings = {
  bot_name: 'DocuBot',
  welcome_message: 'Halo! Ada yang bisa saya bantu?',
  bot_active: true,
  temperature: 0.3,
  max_tokens: 500,
  top_k: 5,
  min_score: 0.3,
}

export function SettingsPage() {
  const [form, setForm] = useState<Settings>(empty)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    settingsApi
      .get()
      .then(setForm)
      .catch((e) => setError(e instanceof HttpError ? e.message : 'gagal memuat setelan'))
  }, [])

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    setSaved('')
    try {
      const next = await settingsApi.put(form)
      setForm(next)
      setSaved('Tersimpan.')
    } catch (err) {
      setError(err instanceof HttpError ? err.message : 'gagal menyimpan')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="max-w-xl">
      <h1 className="text-xl font-semibold text-ink">Setelan</h1>
      <p className="mb-6 text-sm text-muted">Nama bot, welcome message, dan parameter RAG.</p>
      <form className="space-y-4 rounded-xl border border-slate-200 bg-white p-6" onSubmit={onSubmit}>
        <div>
          <Label>Nama bot</Label>
          <Input value={form.bot_name} onChange={(e) => setForm({ ...form, bot_name: e.target.value })} required />
        </div>
        <div>
          <Label>Welcome message</Label>
          <Textarea rows={3} value={form.welcome_message} onChange={(e) => setForm({ ...form, welcome_message: e.target.value })} required />
        </div>
        <label className="flex items-center gap-2 text-sm text-ink">
          <input
            type="checkbox"
            checked={form.bot_active}
            onChange={(e) => setForm({ ...form, bot_active: e.target.checked })}
          />
          Bot aktif
        </label>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <Label>Temperature</Label>
            <Input
              type="number"
              step="0.1"
              min={0}
              max={2}
              value={form.temperature}
              onChange={(e) => setForm({ ...form, temperature: Number(e.target.value) })}
            />
          </div>
          <div>
            <Label>Max tokens</Label>
            <Input
              type="number"
              min={1}
              max={500}
              value={form.max_tokens}
              onChange={(e) => setForm({ ...form, max_tokens: Number(e.target.value) })}
            />
          </div>
          <div>
            <Label>Top-k</Label>
            <Input
              type="number"
              min={1}
              max={20}
              value={form.top_k}
              onChange={(e) => setForm({ ...form, top_k: Number(e.target.value) })}
            />
          </div>
          <div>
            <Label>Min score</Label>
            <Input
              type="number"
              step="0.05"
              min={0}
              max={1}
              value={form.min_score}
              onChange={(e) => setForm({ ...form, min_score: Number(e.target.value) })}
            />
          </div>
        </div>
        {error && <p className="text-sm text-red-600">{error}</p>}
        {saved && <p className="text-sm text-emerald-600">{saved}</p>}
        <Button type="submit" disabled={busy}>
          {busy ? 'Menyimpan...' : 'Simpan'}
        </Button>
      </form>
    </div>
  )
}
