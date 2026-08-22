import { cn } from '../../lib/format'

const styles: Record<string, string> = {
  ready: 'bg-emerald-50 text-emerald-700',
  processing: 'bg-amber-50 text-amber-700',
  pending: 'bg-slate-100 text-slate-600',
  failed: 'bg-red-50 text-red-700',
}

export function Badge({ status }: { status: string }) {
  return (
    <span className={cn('inline-flex rounded-full px-2 py-0.5 text-xs font-medium', styles[status] ?? 'bg-slate-100 text-slate-600')}>
      {status}
    </span>
  )
}
