import { useState } from 'react'
import { Check, Copy } from 'lucide-react'

export function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    await navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <button
      onClick={handleCopy}
      className="p-2 rounded-lg text-[var(--muted)] hover:text-[var(--text)] hover:bg-white/5 transition-colors"
      aria-label="Copy to clipboard"
    >
      {copied ? <Check className="w-4 h-4 text-[var(--green)]" /> : <Copy className="w-4 h-4" />}
    </button>
  )
}
