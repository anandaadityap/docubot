import { useEffect, useRef, useState } from 'react'
import { botApi, streamChat } from '../../api/chat'
import { HttpError } from '../../api/client'
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

const SSE_TIMEOUT_MS = 60_000

type StoredChat = {
  conversationId: string | null
  messages: UIMessage[]
}

export type ChatWindowProps = {
  slug: string
  variant?: 'page' | 'embed'
  channel?: 'public' | 'playground'
}

function sessionKey(slug: string, channel: string) {
  if (channel === 'playground') return `docubot_playground_chat:${slug}`
  return `docubot_public_chat:${slug}`
}

function loadSession(key: string): StoredChat {
  try {
    const raw = sessionStorage.getItem(key)
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

function saveSession(key: string, conversationId: string | null, messages: UIMessage[]) {
  const payload: StoredChat = {
    conversationId,
    messages: messages.map((m) => ({ ...m, streaming: false })),
  }
  sessionStorage.setItem(key, JSON.stringify(payload))
}

export function ChatWindow({ slug, variant = 'page', channel = 'public' }: ChatWindowProps) {
  const key = sessionKey(slug, channel)
  const initial = loadSession(key)
  const [bot, setBot] = useState<PublicBot | null>(null)
  const [notFound, setNotFound] = useState(false)
  const [messages, setMessages] = useState<UIMessage[]>(initial.messages)
  const [input, setInput] = useState('')
  const [conversationId, setConversationId] = useState<string | null>(initial.conversationId)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [retryText, setRetryText] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)
  const abortRef = useRef<AbortController | null>(null)

  useEffect(() => {
    let cancelled = false
    botApi
      .public(slug)
      .then((b) => {
        if (!cancelled) {
          setBot(b)
          setNotFound(false)
        }
      })
      .catch((e) => {
        if (cancelled) return
        if (e instanceof HttpError && e.status === 404) {
          setNotFound(true)
          setBot(null)
          return
        }
        setNotFound(false)
        setBot(null)
      })
    return () => {
      cancelled = true
    }
  }, [slug])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, busy])

  useEffect(() => {
    saveSession(key, conversationId, messages)
  }, [key, conversationId, messages])

  function newChat() {
    abortRef.current?.abort()
    sessionStorage.removeItem(key)
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
        slug,
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
        channel,
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

  if (notFound) {
    return (
      <div className="flex h-full flex-col items-center justify-center bg-surface px-6 text-center">
        <p className="text-lg font-semibold text-ink">Bot tidak ditemukan</p>
        <p className="mt-2 max-w-sm text-sm text-muted">Slug ini tidak terdaftar. Cek tautan pasang di halaman admin, atau buka landing DocuBot.</p>
      </div>
    )
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
  const compact = variant === 'embed'

  return (
    <div className="flex h-full flex-col bg-surface">
      <header
        className={`flex shrink-0 items-center justify-between border-b border-slate-200 bg-white ${compact ? 'px-3 py-2' : 'px-4 py-3'}`}
      >
        <div className="flex items-center gap-2">
          <span className="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-brand text-sm font-semibold text-white">
            {name.slice(0, 1).toUpperCase()}
          </span>
          <div>
            <p className="text-sm font-semibold text-ink">{name}</p>
            <p className="text-[11px] text-muted">{statusLabel}</p>
          </div>
        </div>
        <button type="button" className="text-xs text-muted hover:text-brand" onClick={newChat}>
          Chat baru
        </button>
      </header>

      <div className={`mx-auto flex min-h-0 w-full max-w-2xl flex-1 flex-col ${compact ? 'px-3 py-3' : 'px-4 py-6'}`}>
        <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto">
          <MessageBubble role="bot" content={welcome} />
          {kbEmpty && (
            <p className="text-center text-xs text-amber-700">
              Belum ada dokumen Ready. Pengelola perlu mengunggah `.md` / `.txt` dulu. Pertanyaan tetap bisa dikirim; bot akan
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

      <div className={`shrink-0 border-t border-slate-200 bg-white ${compact ? 'px-3 py-2' : 'px-4 py-3'}`}>
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
