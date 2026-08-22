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
  truncated?: boolean
}

const SESSION_KEY = 'docubot_public_chat'
const SSE_TIMEOUT_MS = 60_000

type StoredChat = {
  conversationId: string | null
  messages: UIMessage[]
}

function loadSession(): StoredChat {
  try {
    const raw = sessionStorage.getItem(SESSION_KEY)
    if (!raw) return { conversationId: null, messages: [] }
    const parsed = JSON.parse(raw) as StoredChat
    return {
      conversationId: parsed.conversationId || null,
      messages: (parsed.messages ?? []).map((m) => ({ ...m, streaming: false })),
    }
  } catch {
    return { conversationId: null, messages: [] }
  }
}

function saveSession(conversationId: string | null, messages: UIMessage[]) {
  const payload: StoredChat = {
    conversationId,
    messages: messages.map((m) => ({ ...m, streaming: false })),
  }
  sessionStorage.setItem(SESSION_KEY, JSON.stringify(payload))
}

export function ChatWindow() {
  const initial = loadSession()
  const [bot, setBot] = useState<PublicBot | null>(null)
  const [messages, setMessages] = useState<UIMessage[]>(initial.messages)
  const [input, setInput] = useState('')
  const [conversationId, setConversationId] = useState<string | null>(initial.conversationId)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [retryText, setRetryText] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)
  const abortRef = useRef<AbortController | null>(null)

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
          has_ready_kb: false,
          register_open: true,
        }),
      )
  }, [])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, busy])

  useEffect(() => {
    saveSession(conversationId, messages)
  }, [conversationId, messages])

  function newChat() {
    abortRef.current?.abort()
    sessionStorage.removeItem(SESSION_KEY)
    setConversationId(null)
    setMessages([])
    setError('')
    setRetryText('')
    setBusy(false)
  }

  async function sendText(text: string) {
    if (!text || busy) return
    setInput('')
    setError('')
    setRetryText('')
    setBusy(true)
    const userMsg: UIMessage = { id: `u-${Date.now()}`, role: 'user', content: text }
    const botId = `b-${Date.now()}`
    setMessages((prev) => [...prev, userMsg, { id: botId, role: 'bot', content: '', streaming: true, sources: [] }])

    const ac = new AbortController()
    abortRef.current = ac
    const timer = window.setTimeout(() => ac.abort(), SSE_TIMEOUT_MS)

    let acc = ''
    let sawDone = false
    try {
      await streamChat(
        text,
        conversationId,
        (ev) => {
          if (ev.type === 'sources') {
            setMessages((prev) => prev.map((m) => (m.id === botId ? { ...m, sources: ev.sources } : m)))
          } else if (ev.type === 'token') {
            acc += ev.content
            setMessages((prev) => prev.map((m) => (m.id === botId ? { ...m, content: acc } : m)))
          } else if (ev.type === 'inactive') {
            acc = ev.message
            setMessages((prev) => prev.map((m) => (m.id === botId ? { ...m, content: acc, streaming: false } : m)))
          } else if (ev.type === 'done') {
            sawDone = true
            setConversationId(ev.conversation_id)
            setMessages((prev) => prev.map((m) => (m.id === botId ? { ...m, streaming: false } : m)))
          } else if (ev.type === 'error') {
            setError(ev.message)
            setRetryText(text)
            setMessages((prev) =>
              prev.map((m) => (m.id === botId ? { ...m, content: acc || ev.message, streaming: false, truncated: Boolean(acc) } : m)),
            )
          }
        },
        ac.signal,
      )
      if (!sawDone && !ac.signal.aborted) {
        setRetryText(text)
        setMessages((prev) =>
          prev.map((m) =>
            m.id === botId ? { ...m, content: acc || 'Jawaban terpotong. Silakan coba lagi.', streaming: false, truncated: true } : m,
          ),
        )
      }
    } catch (e) {
      const aborted = ac.signal.aborted
      const msg = aborted ? 'Waktu habis atau koneksi terputus. Silakan coba lagi.' : e instanceof Error ? e.message : 'gagal mengirim'
      setError(msg)
      setRetryText(text)
      setMessages((prev) =>
        prev.map((m) => (m.id === botId ? { ...m, content: acc || msg, streaming: false, truncated: Boolean(acc) } : m)),
      )
    } finally {
      window.clearTimeout(timer)
      abortRef.current = null
      setBusy(false)
      setMessages((prev) => prev.map((m) => (m.id === botId ? { ...m, streaming: false } : m)))
    }
  }

  const name = bot?.bot_name || 'DocuBot'
  const welcome = bot?.welcome_message || 'Halo! Ada yang bisa saya bantu?'
  const botOff = Boolean(bot && (!bot.configured || !bot.bot_active))
  const kbEmpty = Boolean(bot?.configured && bot.bot_active && !bot.has_ready_kb)
  const statusLabel = !bot
    ? 'Memuat...'
    : !bot.configured
      ? 'Belum dikonfigurasi'
      : !bot.bot_active
        ? 'Tidak aktif'
        : !bot.has_ready_kb
          ? 'Online · knowledge base kosong'
          : 'Online'
  const inputDisabled = busy || botOff
  const placeholder = botOff
    ? bot?.configured
      ? 'Bot sedang tidak aktif'
      : 'Bot belum dikonfigurasi'
    : 'Ketik pertanyaan...'

  return (
    <div className="flex h-full flex-col bg-surface">
      <header className="flex shrink-0 items-center justify-between border-b border-slate-200 bg-white px-4 py-3">
        <div className="flex items-center gap-2">
          <span className="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-brand text-sm font-semibold text-white">
            {name.slice(0, 1).toUpperCase()}
          </span>
          <div>
            <p className="text-sm font-semibold text-ink">{name}</p>
            <p className="text-[11px] text-muted">{statusLabel}</p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <button type="button" className="text-xs text-muted hover:text-brand" onClick={newChat}>
            Chat baru
          </button>
          <Link to="/login" className="text-xs text-muted hover:text-brand">
            Admin Login
          </Link>
        </div>
      </header>

      <div className="mx-auto flex min-h-0 w-full max-w-2xl flex-1 flex-col px-4 py-6">
        <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto">
          <MessageBubble role="bot" content={welcome} />
          {!bot?.configured && (
            <p className="text-center text-xs text-muted">
              Belum ada admin.{' '}
              {bot?.register_open !== false ? (
                <Link to="/register" className="text-brand hover:underline">
                  Daftar
                </Link>
              ) : (
                'Pendaftaran sudah ditutup.'
              )}{' '}
              lalu unggah dokumen knowledge base.
            </p>
          )}
          {kbEmpty && (
            <p className="text-center text-xs text-amber-700">
              Belum ada dokumen Ready. Admin perlu mengunggah `.md` / `.txt` dulu. Pertanyaan tetap bisa dikirim; bot akan
              menjawab jujur bahwa knowledge base kosong.
            </p>
          )}
          {messages.map((m) => (
            <MessageBubble
              key={m.id}
              role={m.role}
              content={m.content}
              sources={m.sources}
              streaming={m.streaming}
              truncated={m.truncated}
            />
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
            void sendText(input.trim())
          }}
        >
          <input
            className="flex-1 rounded-xl border border-slate-200 px-4 py-2.5 text-sm outline-none placeholder:text-slate-400 focus:border-brand focus:ring-2 focus:ring-brand/20 disabled:bg-slate-50"
            placeholder={placeholder}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            disabled={inputDisabled}
            maxLength={2000}
          />
          <Button type="submit" disabled={inputDisabled || !input.trim()}>
            Kirim
          </Button>
        </form>
        {retryText && !busy && (
          <p className="mx-auto mt-2 max-w-2xl text-center text-xs">
            <button type="button" className="text-brand hover:underline" onClick={() => void sendText(retryText)}>
              Coba lagi
            </button>
          </p>
        )}
        {error && <p className="mx-auto mt-2 max-w-2xl text-center text-xs text-red-600">{error}</p>}
        <p className="mx-auto mt-2 max-w-2xl text-center text-[11px] text-slate-400">
          Jawaban bot bisa salah. Cek sumber kutipan.
        </p>
      </div>
    </div>
  )
}
