import { type ReactNode, useState } from 'react'
import type { ChatSource } from '../../api/types'
import { cn } from '../../lib/format'

export function SourceChip({ index, source }: { index: number; source?: ChatSource }) {
  const [open, setOpen] = useState(false)
  if (!source) {
    return <span className="text-brand font-medium">[{index}]</span>
  }
  return (
    <span className="relative inline-block">
      <button
        type="button"
        className="mx-0.5 inline-flex h-5 min-w-5 items-center justify-center rounded-md bg-brand/10 px-1 text-[11px] font-semibold text-brand hover:bg-brand/20"
        onClick={() => setOpen((v) => !v)}
        aria-label={`Sumber ${index}: ${source.filename}`}
      >
        {index}
      </button>
      {open && (
        <>
          <button type="button" className="fixed inset-0 z-10 cursor-default" aria-label="Tutup" onClick={() => setOpen(false)} />
          <div className="absolute bottom-full left-1/2 z-20 mb-2 w-64 -translate-x-1/2 rounded-xl border border-slate-200 bg-white p-3 text-left shadow-lg">
            <p className="truncate text-xs font-semibold text-ink">{source.filename}</p>
            <p className="mt-1 text-xs leading-relaxed text-muted">{source.snippet}</p>
            <p className="mt-2 text-[11px] text-slate-400">skor relevansi {source.score.toFixed(2)}</p>
          </div>
        </>
      )}
    </span>
  )
}

export function MessageBubble({
  role,
  content,
  sources,
  streaming,
  truncated,
}: {
  role: 'user' | 'bot'
  content: string
  sources?: ChatSource[]
  streaming?: boolean
  truncated?: boolean
}) {
  const isUser = role === 'user'
  const showSources = !isUser && Boolean(sources?.length) && !streaming
  return (
    <div className={cn('flex', isUser ? 'justify-end' : 'justify-start')}>
      <div
        className={cn(
          'max-w-[85%] rounded-2xl px-4 py-2.5 text-sm leading-relaxed',
          isUser ? 'bg-brand text-white rounded-br-md' : 'bg-white border border-slate-200 text-ink rounded-bl-md shadow-sm',
        )}
      >
        {isUser ? (
          <p className="whitespace-pre-wrap">{content}</p>
        ) : (
          <div>
            <BotRichText text={content} sources={sources} />
            {streaming && <span className="ml-0.5 inline-block h-3 w-0.5 animate-pulse bg-brand align-middle" />}
            {truncated && <p className="mt-1 text-[11px] text-amber-700">Jawaban mungkin terpotong.</p>}
            {showSources && (
              <div className="mt-2 flex flex-wrap gap-1 border-t border-slate-100 pt-2">
                <span className="mr-1 text-[11px] text-muted">Sumber:</span>
                {sources!.map((s, i) => (
                  <SourceChip key={`${s.doc_id}-${i}`} index={i + 1} source={s} />
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

function BotRichText({ text, sources }: { text: string; sources?: ChatSource[] }) {
  const lines = text.replace(/\r\n/g, '\n').split('\n')
  const blocks: ReactNode[] = []
  let list: string[] = []
  let key = 0

  const flushList = () => {
    if (list.length === 0) return
    const items = list
    list = []
    blocks.push(
      <ul key={`ul-${key++}`} className="my-1.5 list-disc space-y-1 pl-5">
        {items.map((item, i) => (
          <li key={i}>{renderInline(item, sources)}</li>
        ))}
      </ul>,
    )
  }

  for (const line of lines) {
    const bullet = line.match(/^\s*[-*]\s+(.+)$/)
    if (bullet) {
      list.push(bullet[1])
      continue
    }
    flushList()
    const trimmed = line.trim()
    if (trimmed === '') {
      continue
    }
    blocks.push(
      <p key={`p-${key++}`} className="mb-1.5 last:mb-0">
        {renderInline(trimmed, sources)}
      </p>,
    )
  }
  flushList()

  if (blocks.length === 0) {
    return <p>{renderInline(text, sources)}</p>
  }
  return <>{blocks}</>
}

const inlineRe = /(\[\d+\]|\*\*[^*]+?\*\*|`[^`]+?`)/g

function renderInline(text: string, sources?: ChatSource[]): ReactNode[] {
  const parts = text.split(inlineRe)
  return parts.map((part, i) => {
    if (!part) return null
    const cite = part.match(/^\[(\d+)\]$/)
    if (cite) {
      const idx = Number(cite[1])
      return <SourceChip key={i} index={idx} source={sources?.[idx - 1]} />
    }
    if (part.startsWith('**') && part.endsWith('**') && part.length >= 4) {
      return (
        <strong key={i} className="font-semibold">
          {part.slice(2, -2)}
        </strong>
      )
    }
    if (part.startsWith('`') && part.endsWith('`') && part.length >= 2) {
      return (
        <code key={i} className="rounded bg-slate-100 px-1 py-0.5 font-mono text-[12px] text-slate-800">
          {part.slice(1, -1)}
        </code>
      )
    }
    return <span key={i}>{part}</span>
  })
}

export function TypingIndicator() {
  return (
    <div className="flex justify-start">
      <div className="rounded-2xl rounded-bl-md border border-slate-200 bg-white px-4 py-3 shadow-sm">
        <span className="flex gap-1">
          <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-slate-400 [animation-delay:-0.2s]" />
          <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-slate-400 [animation-delay:-0.1s]" />
          <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-slate-400" />
        </span>
      </div>
    </div>
  )
}
