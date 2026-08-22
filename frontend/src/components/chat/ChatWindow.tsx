import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { botApi, streamChat } from '../../api/chat'
import type { ChatSource, PublicBot } from '../../api/types'
import { Button } from '../ui/Button'
import { MessageBubble, TypingIndicator } from './MessageBubble'

type UIMessage = {
  id: string
  role: 'user' | 'bot'
  content: string
  sources?: ChatSource[]
  streaming?: boolean
}

export function ChatWindow() {
  const [bot, setBot] = useState<PublicBot | null>(null)
  const [messages, setMessages] = useState<UIMessage[]>([])
  const [input, setInput] = useState('')
  const [conversationId, setConversationId] = useState<number | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    botApi
      .public()
      .then(setBot)
      .catch(() =>
        setBot({
          bot_name: 'DocuBot',
          welcome_message: 'Halo! Ada yang bisa saya bantu?',
          bot_active: false,
          configured: false,
        }),
      )
  }, [])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, busy])

  async function send() {
    const text = input.trim()
    if (!text || busy) return
    setInput('')
    setError('')
    setBusy(true)
    const userMsg: UIMessage = { id: `u-${Date.now()}`, role: 'user', content: text }
    const botId = `b-${Date.now()}`
    setMessages((prev) => [...prev, userMsg, { id: botId, role: 'bot', content: '', streaming: true, sources: [] }])

    let acc = ''
    try {
      await streamChat(text, conversationId, (ev) => {
        if (ev.type === 'sources') {
          setMessages((prev) => prev.map((m) => (m.id === botId ? { ...m, sources: ev.sources } : m)))
        } else if (ev.type === 'token') {
          acc += ev.content
          setMessages((prev) => prev.map((m) => (m.id === botId ? { ...m, content: acc } : m)))
        } else if (ev.type === 'inactive') {
          acc = ev.message
          setMessages((prev) => prev.map((m) => (m.id === botId ? { ...m, content: acc, streaming: false } : m)))
        } else if (ev.type === 'done') {
          setConversationId(ev.conversation_id)
          setMessages((prev) => prev.map((m) => (m.id === botId ? { ...m, streaming: false } : m)))
        } else if (ev.type === 'error') {
          setError(ev.message)
          setMessages((prev) => prev.map((m) => (m.id === botId ? { ...m, content: acc || ev.message, streaming: false } : m)))
        }
      })
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'gagal mengirim'
      setError(msg)
      setMessages((prev) => prev.map((m) => (m.id === botId ? { ...m, content: acc || msg, streaming: false } : m)))
    } finally {
      setBusy(false)
      setMessages((prev) => prev.map((m) => (m.id === botId ? { ...m, streaming: false } : m)))
    }
  }

  const name = bot?.bot_name || 'DocuBot'
  const welcome = bot?.welcome_message || 'Halo! Ada yang bisa saya bantu?'
  const inactive = bot && (!bot.configured || !bot.bot_active)

  return (
    <div className="flex h-full flex-col bg-surface">
      <header className="flex shrink-0 items-center justify-between border-b border-slate-200 bg-white px-4 py-3">
        <div className="flex items-center gap-2">
          <span className="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-brand text-sm font-semibold text-white">
            {name.slice(0, 1).toUpperCase()}
          </span>
          <div>
            <p className="text-sm font-semibold text-ink">{name}</p>
            <p className="text-[11px] text-muted">{inactive ? 'Tidak aktif' : 'Online'}</p>
          </div>
        </div>
        <Link to="/login" className="text-xs text-muted hover:text-brand">
          Admin Login
        </Link>
      </header>

      <div className="mx-auto flex min-h-0 w-full max-w-2xl flex-1 flex-col px-4 py-6">
        <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto">
          <MessageBubble role="bot" content={welcome} />
          {!bot?.configured && (
            <p className="text-center text-xs text-muted">
              Belum ada admin. <Link to="/register" className="text-brand hover:underline">Daftar</Link> lalu unggah dokumen knowledge base.
            </p>
          )}
          {messages.map((m) => (
            <MessageBubble key={m.id} role={m.role} content={m.content} sources={m.sources} streaming={m.streaming} />
          ))}
          {busy && messages[messages.length - 1]?.content === '' && <TypingIndicator />}
          <div ref={bottomRef} />
        </div>
      </div>

      <div className="shrink-0 border-t border-slate-200 bg-white px-4 py-3">
        <form
          className="mx-auto flex max-w-2xl gap-2"
          onSubmit={(e) => {
            e.preventDefault()
            void send()
          }}
        >
          <input
            className="flex-1 rounded-xl border border-slate-200 px-4 py-2.5 text-sm outline-none placeholder:text-slate-400 focus:border-brand focus:ring-2 focus:ring-brand/20"
            placeholder="Ketik pertanyaan..."
            value={input}
            onChange={(e) => setInput(e.target.value)}
            disabled={busy}
            maxLength={2000}
          />
          <Button type="submit" disabled={busy || !input.trim()}>
            Kirim
          </Button>
        </form>
        {error && <p className="mx-auto mt-2 max-w-2xl text-center text-xs text-red-600">{error}</p>}
        <p className="mx-auto mt-2 max-w-2xl text-center text-[11px] text-slate-400">
          Jawaban bot bisa salah. Cek sumber kutipan.
        </p>
      </div>
    </div>
  )
}
