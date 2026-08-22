import type { ButtonHTMLAttributes } from 'react'
import { cn } from '../../lib/format'

type Props = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: 'primary' | 'ghost' | 'danger' | 'outline'
}

export function Button({ variant = 'primary', className, ...props }: Props) {
  const styles = {
    primary: 'bg-brand text-white hover:bg-brand-dark disabled:opacity-50',
    ghost: 'text-slate-600 hover:bg-slate-100 disabled:opacity-50',
    danger: 'text-red-600 hover:bg-red-50 disabled:opacity-50',
    outline: 'border border-slate-200 bg-white text-ink hover:bg-slate-50 disabled:opacity-50',
  }[variant]
  return (
    <button
      className={cn(
        'inline-flex items-center justify-center gap-2 rounded-lg px-3.5 py-2 text-sm font-medium transition',
        styles,
        className,
      )}
      {...props}
    />
  )
}
