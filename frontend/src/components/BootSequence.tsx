import { useEffect, useRef, useState } from 'react'
import { clsx } from 'clsx'
import { Radio } from 'lucide-react'

const bootLines = [
  { text: 'geoctl bootloader v1.0.0', delay: 80 },
  { text: 'Platform: linux/amd64 · Go 1.25 · static binary', delay: 120 },
  { text: '', delay: 40 },
  { text: 'Loading gitlab-geo-sync control plane...', delay: 180 },
  { text: '[ OK ] Mounted /var/opt/gitlab (read-only)', delay: 110 },
  { text: '[ OK ] Started PostgreSQL WAL receiver', delay: 130 },
  { text: '[ OK ] Started sshsync transport layer', delay: 110 },
  { text: '[ OK ] Started reconciler runner (interval=5m)', delay: 140 },
  { text: '[ OK ] Started Prometheus metrics exporter', delay: 120 },
  { text: '[ OK ] Started webhook receiver on :8080', delay: 110 },
  { text: '', delay: 40 },
  { text: 'geoctl v1.0.0 — clean-room geo-replication engine', delay: 200 },
  { text: 'License: Apache-2.0', delay: 100 },
  { text: '', delay: 60 },
  { text: 'Handshake with primary... ESTABLISHED', delay: 160, pulse: true },
  { text: 'Secondary node ready. Click to enter dashboard.', delay: 240, final: true },
]

export function BootSequence({ onDone }: { onDone: () => void }) {
  const [visibleCount, setVisibleCount] = useState(0)
  const [dismissed, setDismissed] = useState(false)
  const [autoDone, setAutoDone] = useState(false)
  const doneRef = useRef(false)

  useEffect(() => {
    let totalDelay = 0
    const timers: ReturnType<typeof setTimeout>[] = []

    bootLines.forEach((line, index) => {
      totalDelay += line.delay
      const timer = setTimeout(() => {
        if (doneRef.current) return
        setVisibleCount(index + 1)
        if (index === bootLines.length - 1) {
          setAutoDone(true)
          const doneTimer = setTimeout(() => {
            if (!doneRef.current) {
              doneRef.current = true
              setDismissed(true)
              setTimeout(onDone, 350)
            }
          }, 1400)
          timers.push(doneTimer)
        }
      }, totalDelay)
      timers.push(timer)
    })

    return () => timers.forEach(clearTimeout)
  }, [onDone])

  useEffect(() => {
    const keyHandler = () => {
      if (!autoDone) return
      if (doneRef.current) return
      doneRef.current = true
      setDismissed(true)
      setTimeout(onDone, 350)
    }
    window.addEventListener('keydown', keyHandler)
    return () => window.removeEventListener('keydown', keyHandler)
  }, [autoDone, onDone])

  const handleClick = () => {
    if (!autoDone) return
    if (doneRef.current) return
    doneRef.current = true
    setDismissed(true)
    setTimeout(onDone, 350)
  }

  return (
    <div
      className={clsx(
        'fixed inset-0 z-[100] bg-[var(--bg)] flex flex-col items-center justify-center p-6 cursor-pointer select-none',
        dismissed && 'opacity-0 pointer-events-none'
      )}
      style={{ transition: 'opacity 0.35s ease' }}
      onClick={handleClick}
    >
      <div className="absolute inset-0 aurora" />

      <div className="relative w-full max-w-3xl">
        <div className="mb-8 flex items-center gap-4">
          <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-[var(--accent)] to-[var(--teal)] flex items-center justify-center shadow-lg shadow-[var(--accent-glow)] animate-pulse">
            <Radio className="w-5 h-5 text-white" />
          </div>
          <div>
            <div className="font-display font-bold tracking-widest text-sm text-[var(--accent-light)]">
              GITLAB-GEO-SYNC
            </div>
            <div className="text-[var(--muted)] text-xs">Control Plane Bootloader v1.0.0</div>
          </div>
        </div>

        <div className="rounded-2xl border border-[var(--border)] bg-[var(--surface)]/80 backdrop-blur p-6 md:p-8 font-mono text-sm leading-7 shadow-2xl">
          <div className="text-[var(--accent-light)] mb-4 pb-3 border-b border-[var(--border)] flex items-center gap-2">
            <span className="text-[var(--teal)]">geoctl</span>
            <span className="text-[var(--muted)]">@secondary-node:~$</span>
            <span className="text-[var(--text)]">boot --verbose</span>
          </div>

          <div className="space-y-0.5">
            {bootLines.map((line, i) => (
              <div
                key={i}
                className={clsx(
                  'transition-opacity duration-75',
                  i < visibleCount ? 'opacity-100' : 'opacity-0',
                  line.final ? 'text-[var(--accent-light)] mt-4 font-semibold' : '',
                  line.pulse ? 'text-[var(--teal)] font-semibold' : ''
                )}
              >
                {line.text === '' ? (
                  <span> </span>
                ) : line.text.startsWith('[ OK ]') ? (
                  <>
                    <span className="text-[var(--green)]">[ OK ]</span>
                    <span className="text-[var(--muted)]"> {line.text.slice(6)}</span>
                  </>
                ) : (
                  <span className={line.final || line.pulse ? 'text-[var(--text)]' : 'text-[var(--muted)]'}>
                    {line.text}
                  </span>
                )}
              </div>
            ))}
            <div
              className={clsx(
                'inline-block w-2.5 h-5 bg-[var(--accent)] mt-3',
                visibleCount >= bootLines.length ? 'animate-pulse' : 'opacity-0'
              )}
              style={{ verticalAlign: 'text-bottom' }}
            />
          </div>
        </div>

        <div className="mt-6 text-center text-[var(--muted)] text-xs animate-pulse">
          {visibleCount < bootLines.length ? 'Booting control plane...' : 'Click or press any key to continue'}
        </div>
      </div>
    </div>
  )
}
