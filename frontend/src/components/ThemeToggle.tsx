import { useEffect, useState } from 'react'
import { Moon, Sun } from 'lucide-react'

export function ThemeToggle() {
  const [theme, setTheme] = useState<'dark' | 'light'>('dark')

  useEffect(() => {
    const saved = localStorage.getItem('ggs-theme') as 'dark' | 'light' | null
    const initial = saved || 'dark'
    setTheme(initial)
    document.documentElement.classList.toggle('light', initial === 'light')
  }, [])

  const toggle = () => {
    const next = theme === 'dark' ? 'light' : 'dark'
    setTheme(next)
    document.documentElement.classList.toggle('light', next === 'light')
    localStorage.setItem('ggs-theme', next)
  }

  return (
    <button
      onClick={toggle}
      className="p-2 rounded-lg border border-[var(--border)] bg-[var(--surface)]/60 text-[var(--muted)] hover:text-[var(--text)] hover:border-[var(--accent)]/50 transition-colors"
      aria-label="Toggle theme"
    >
      {theme === 'dark' ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
    </button>
  )
}
