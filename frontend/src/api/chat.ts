import { api } from './client'
import type { ChatSource } from './types'

export type ChatEvent =
  | { type: 'sources'; sources: ChatSource[] }
  | { type: 'token'; content: string }
  | { type: 'inactive'; message: string }
  | { type: 'done'; conversation_id: number; message_id: number; total_tokens: number; latency_ms: number }
  | { type: 'error'; code: string; message: string }

function parseBlock(block: string): ChatEvent | null {
  let event = ''
  let data = ''
  for (const line of block.split('\n')) {
    if (line.startsWith('event:')) event = line.slice(6).trim()
    else if (line.startsWith('data:')) data += line.slice(5).trim()
  }
  if (!event || !data) return null
  try {
    const payload = JSON.parse(data) as Record<string, unknown>
    if (event === 'sources') {
      return { type: 'sources', sources: (payload.sources as ChatSource[]) ?? [] }
    }
    if (event === 'token') {
      return { type: 'token', content: String(payload.content ?? '') }
    }
    if (event === 'inactive') {
      return { type: 'inactive', message: String(payload.message ?? '') }
    }
    if (event === 'done') {
      return {
        type: 'done',
        conversation_id: Number(payload.conversation_id),
        message_id: Number(payload.message_id),
        total_tokens: Number(payload.total_tokens ?? 0),
        latency_ms: Number(payload.latency_ms ?? 0),
      }
    }
    if (event === 'error') {
      return { type: 'error', code: String(payload.code ?? 'ERROR'), message: String(payload.message ?? 'error') }
    }
  } catch {
    return null
  }
  return null
}

export async function streamChat(
  message: string,
  conversationId: number | null,
  onEvent: (ev: ChatEvent) => void,
  signal?: AbortSignal,
) {
  const res = await fetch('/api/v1/chat', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      message,
      conversation_id: conversationId,
    }),
    signal,
  })
  if (!res.ok) {
    let msg = res.statusText
    try {
      const body = (await res.json()) as { error?: { message: string; code: string } }
      if (body.error) {
        onEvent({ type: 'error', code: body.error.code, message: body.error.message })
        return
      }
    } catch {
      msg = await res.text()
    }
    onEvent({ type: 'error', code: 'HTTP_ERROR', message: msg || 'gagal mengirim pesan' })
    return
  }
  if (!res.body) {
    onEvent({ type: 'error', code: 'HTTP_ERROR', message: 'streaming tidak didukung' })
    return
  }

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buf = ''
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buf += decoder.decode(value, { stream: true })
    const parts = buf.split('\n\n')
    buf = parts.pop() ?? ''
    for (const block of parts) {
      const ev = parseBlock(block.trim())
      if (ev) onEvent(ev)
    }
  }
  if (buf.trim()) {
    const ev = parseBlock(buf.trim())
    if (ev) onEvent(ev)
  }
}

export const botApi = {
  public: () => api<import('./types').PublicBot>('/api/v1/bot'),
}
