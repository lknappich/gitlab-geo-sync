import { useEffect, useState } from 'react'
import {
  Activity,
  ArrowRight,
  Boxes,
  CheckCircle2,
  ChevronRight,
  Database,
  GitBranch,
  Globe,
  HardDrive,
  Radio,
  Server,
  ShieldCheck,
  Terminal,
} from 'lucide-react'
import { clsx } from 'clsx'
import { BootSequence } from './components/BootSequence.tsx'
import { CopyButton } from './components/CopyButton.tsx'
import { ThemeToggle } from './components/ThemeToggle.tsx'

const projectMeta = [
  { label: 'Latest release', value: 'v1.0.0', note: '[Unreleased]' },
  { label: 'Go toolchain', value: '1.25', note: 'Required' },
  { label: 'License', value: 'Apache-2.0', note: 'Open source' },
  { label: 'Distribution', value: 'Single static binary', note: 'cmd/geoctl' },
  { label: 'Platform target', value: 'Linux / x86_64', note: 'Primary runtime' },
  { label: 'GitLab flavour', value: 'GitLab CE / Omnibus', note: 'Self-hosted' },
  { label: 'Module path', value: 'github.com/anomalyco/gitlab-geo-sync', note: 'Go module' },
]

const capabilities = [
  {
    id: 'database',
    title: 'PostgreSQL WAL streaming',
    tagline: 'Byte-identical database replicas',
    summary: 'Use native PostgreSQL physical streaming replication to copy every row, index, sequence, and encrypted column exactly as it exists on the primary.',
    bullets: [
      'Seed secondaries with pg_basebackup',
      'Continuous WAL streaming with lag monitoring',
      'No application-level transformation or filtering',
    ],
  },
  {
    id: 'git',
    title: 'Git data replication',
    tagline: 'Repositories copied completely',
    summary: 'Replicate the repository tree either through direct filesystem rsync or bounded-parallel git fetch across all projects.',
    bullets: [
      'rsync --delete --checksum for byte parity',
      'Per-project git fetch with configurable worker pools',
      'Project list driven from the replicated GitLab database',
    ],
  },
  {
    id: 'objects',
    title: 'Object storage parity',
    tagline: 'Uploads, artifacts, LFS, and registry',
    summary: 'Cover both cloud-backed and disk-backed object storage without vendor lock-in or proprietary APIs.',
    bullets: [
      'Verify S3 cross-region replication by object count and size',
      'rsync uploads, artifacts, LFS, packages, and registry blobs',
      'Surface drift before it becomes an outage',
    ],
  },
  {
    id: 'registry',
    title: 'Container registry validation',
    tagline: 'Catalog and tag-level diff',
    summary: 'Compare Docker Registry v2 catalogs and tags between sites to confirm container images are present on the secondary.',
    bullets: [
      'Public Registry v2 API only',
      'Gracefully skips registries that require auth',
      'Reports missing repositories and tags',
    ],
  },
  {
    id: 'failover',
    title: 'Controlled failover',
    tagline: 'Promote a secondary safely',
    summary: 'Automated health checks detect primary failure, but promotion stays human-gated by default so operations teams remain in control.',
    bullets: [
      'Configurable failure threshold and quorum',
      'Step-by-step promotion: stop, promote, verify, start',
      'Role-swap the old primary back to secondary afterwards',
    ],
  },
  {
    id: 'observability',
    title: 'Operations and visibility',
    tagline: 'Metrics, logs, and runbooks',
    summary: 'Treat geo-replication like any other production service: measure it, alert on it, and document how to recover from it.',
    bullets: [
      'Prometheus metrics and structured logs',
      'Webhook-driven near-real-time per-project sync',
      'RPO/RTO reports and generated Markdown runbooks',
    ],
  },
]

const architecture = [
  {
    icon: Database,
    title: 'PostgreSQL WAL streaming',
    body: 'Primary replication path. pg_basebackup seeds the secondary; WAL receiver keeps it byte-identical.',
    span: 'md:col-span-2',
  },
  {
    icon: GitBranch,
    title: 'Git data',
    body: 'Choose rsync of /var/opt/gitlab/git-data/repositories or per-project git fetch driven by the replicated DB.',
  },
  {
    icon: Globe,
    title: 'Object storage',
    body: 'Cloud provider cross-region replication for S3, or rsync for disk-backed uploads/artifacts/LFS/registry blobs.',
  },
  {
    icon: ShieldCheck,
    title: 'Encrypted columns',
    body: 'Secondary shares the primary db_key_base so the GitLab app itself can decrypt. geoctl only compares key bytes.',
  },
  {
    icon: Server,
    title: 'Control plane',
    body: 'geoctl orchestrates reconcilers, metrics, webhook receiver, failover loop, and runbook generation.',
  },
  {
    icon: Activity,
    title: 'Observability',
    body: 'Prometheus metrics, structured logs, health checks, and SLA reports for RPO/RTO.',
  },
]

const commands = [
  { cmd: 'geoctl init', desc: 'Interactive config wizard' },
  { cmd: 'geoctl config-validate', desc: 'Load and validate config' },
  { cmd: 'geoctl doctor', desc: 'Check prerequisites' },
  { cmd: 'geoctl pg setup', desc: 'Bootstrap secondary via pg_basebackup' },
  { cmd: 'geoctl pg status', desc: 'Show replication lag' },
  { cmd: 'geoctl sync', desc: 'Run one reconciliation sweep' },
  { cmd: 'geoctl dbkey', desc: 'Verify db_key_base parity' },
  { cmd: 'geoctl failover', desc: 'Promote secondary (human-gated)' },
  { cmd: 'geoctl adopt-as-secondary', desc: 'Role-swap old primary' },
  { cmd: 'geoctl runbook', desc: 'Generate operational runbook' },
  { cmd: 'geoctl sla', desc: 'Print RPO/RTO summary' },
  { cmd: 'geoctl serve', desc: 'Run sync engine + metrics + webhook' },
]

const quickstart = `# 1. Generate a config via the interactive wizard
geoctl init -o config.yaml

# 2. Export secrets (env-only)
export PG_CTRL_PASSWORD=...
export PG_REPL_PASSWORD=...
export SEC_REPL_PASSWORD=...
export S3_AK=...
export S3_SK=...

# 3. Validate and check prerequisites
geoctl config-validate -c config.yaml
geoctl doctor -c config.yaml

# 4. Bootstrap the secondary PostgreSQL
geoctl pg setup --secondary <name> --data-dir /var/opt/gitlab/postgresql/data

# 5. Verify db_key_base parity
geoctl dbkey -c config.yaml

# 6. Start the sync engine
geoctl serve -c config.yaml`

const changelog = [
  { tag: 'Fixed', text: 'PostgreSQL TLS defaults to sslmode=require; libpq-safe password quoting.' },
  { tag: 'Added', text: 'Bounded-parallel git fetch worker pool (default 8) for large instances.' },
  { tag: 'Added', text: 'SECURITY.md, CODE_OF_CONDUCT.md, Dependabot, golangci-lint CI.' },
  { tag: 'Changed', text: 'SLA report fields renamed to honest PGLagCurrent / PGLagPeak.' },
]

function Reveal({ children, delay = 0, className }: { children: React.ReactNode; delay?: number; className?: string }) {
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    const t = setTimeout(() => setVisible(true), delay)
    return () => clearTimeout(t)
  }, [delay])

  return (
    <div
      className={clsx('transition-all duration-700 ease-out', className)}
      style={{
        opacity: visible ? 1 : 0,
        transform: visible ? 'translateY(0)' : 'translateY(20px)',
      }}
    >
      {children}
    </div>
  )
}

function SectionTag({ children, color = 'indigo' }: { children: React.ReactNode; color?: 'indigo' | 'teal' | 'amber' }) {
  const map = {
    indigo: 'border-[var(--accent)]/30 text-[var(--accent-light)] bg-[var(--accent)]/10',
    teal: 'border-[var(--teal)]/30 text-[var(--teal)] bg-[var(--teal)]/10',
    amber: 'border-[var(--amber)]/30 text-[var(--amber)] bg-[var(--amber)]/10',
  }
  return (
    <span className={clsx('section-tag', map[color])}>
      {children}
    </span>
  )
}

export default function App() {
  const [booted, setBooted] = useState(false)
  const [activeCap, setActiveCap] = useState(capabilities[0].id)

  return (
    <div className="relative min-h-screen overflow-hidden">
      <div className="aurora" />

      {!booted && <BootSequence onDone={() => setBooted(true)} />}

      <div className="relative z-10">
        {/* Nav */}
        <nav className="sticky top-0 z-50 border-b border-[var(--border)] glass">
          <div className="max-w-6xl mx-auto px-6 h-16 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-[var(--accent)] to-[var(--teal)] flex items-center justify-center shadow-lg">
                <Radio className="w-4 h-4 text-white" />
              </div>
              <span className="font-display font-bold tracking-tight text-lg text-[var(--text)]">gitlab-geo-sync</span>
            </div>
            <div className="hidden md:flex items-center gap-8 text-sm font-medium text-[var(--muted)]">
              <a href="#overview" className="hover:text-[var(--text)] transition-colors">Overview</a>
              <a href="#capabilities" className="hover:text-[var(--text)] transition-colors">Capabilities</a>
              <a href="#architecture" className="hover:text-[var(--text)] transition-colors">Architecture</a>
            <a href="#commands" className="hover:text-[var(--text)] transition-colors">CLI</a>
            <a href="#docs" className="hover:text-[var(--text)] transition-colors">Docs</a>
            <a
              href="https://github.com/lknappich/gitlab-geo-sync"
              className="hover:text-[var(--text)] transition-colors"
            >
              GitHub
            </a>
          </div>
          <div className="flex items-center gap-3">
            <ThemeToggle />
            <a
              href="#quickstart"
              className="hidden sm:inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-[var(--accent)] text-white text-sm font-semibold hover:bg-[var(--accent-light)] transition-colors shadow-lg shadow-[var(--accent-glow)]"
            >
              Get started
              <ArrowRight className="w-4 h-4" />
            </a>
          </div>
        </div>
      </nav>

        {/* Hero */}
        <header className="relative border-b border-[var(--border)]">
          <div className="max-w-6xl mx-auto px-6 py-24 md:py-32">
            <Reveal>
              <SectionTag color="teal">
                <CheckCircle2 className="w-3.5 h-3.5" />
                Open-source disaster recovery for self-hosted GitLab
              </SectionTag>
            </Reveal>

            <Reveal delay={100}>
              <h1 className="mt-8 text-5xl md:text-7xl font-display font-bold leading-[1.05] max-w-4xl">
                Geo-replicate your
                <br />
                <span className="gradient-text">GitLab instance.</span>
              </h1>
            </Reveal>

            <Reveal delay={200}>
              <p className="mt-6 text-lg md:text-xl text-[var(--muted)] max-w-2xl leading-relaxed">
                A clean-room, Apache-2.0 orchestrator that builds true one-to-one GitLab replicas
                using PostgreSQL WAL streaming, rsync, git fetch, and bucket replication.
              </p>
            </Reveal>

            <Reveal delay={300}>
              <div className="mt-10 flex flex-wrap items-center gap-4">
                <a
                  href="#quickstart"
                  className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-[var(--accent)] text-white font-semibold hover:bg-[var(--accent-light)] transition-all shadow-xl shadow-[var(--accent-glow)]"
                >
                  Quickstart
                  <ChevronRight className="w-4 h-4" />
                </a>
                <a
                  href="https://github.com/lknappich/gitlab-geo-sync"
                  className="inline-flex items-center gap-2 px-6 py-3 rounded-xl border border-[var(--border-strong)] bg-[var(--surface-elevated)] hover:border-[var(--accent)]/50 transition-all"
                >
                  View source
                  <ArrowRight className="w-4 h-4" />
                </a>
              </div>
            </Reveal>

            <Reveal delay={500}>
              <div className="mt-16 grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
                {projectMeta.map((m) => (
                  <div
                    key={m.label}
                    className="p-4 rounded-xl border border-[var(--border)] bg-[var(--surface)]/60 hover:border-[var(--accent)]/30 hover:bg-[var(--surface-elevated)] transition-all"
                  >
                    <div className="text-[var(--muted)] text-[10px] font-bold uppercase tracking-wider mb-1.5">{m.label}</div>
                    <div className="text-[var(--text)] font-semibold text-sm truncate" title={m.value}>{m.value}</div>
                    <div className="text-[var(--accent-light)] text-[10px] mt-1">{m.note}</div>
                  </div>
                ))}
              </div>
            </Reveal>
          </div>
        </header>

        <main className="max-w-6xl mx-auto px-6 py-24 space-y-32">
          {/* Overview / intro */}
          <section id="overview">
            <Reveal>
              <div className="grid md:grid-cols-2 gap-12 items-center">
                <div className="space-y-6">
                  <SectionTag color="indigo">Why gitlab-geo-sync?</SectionTag>
                  <h2 className="text-3xl md:text-4xl font-display font-bold">Built for teams that can’t rely on a paid feature.</h2>
                  <p className="text-[var(--muted)] leading-relaxed">
                    GitLab Geo is a Premium/Ultimate feature. If you operate your own GitLab servers and need
                    a true 1:1 replica for disaster recovery, read-scaling, or migration, this project
                    orchestrates the same standard tools you already use — with observability, idempotent
                    reconciliation, and a safe failover flow.
                  </p>
                  <ul className="space-y-3 text-sm text-[var(--muted)]">
                    {[
                      'No GitLab EE code or internal APIs',
                      'Secrets stay in environment variables only',
                      'Every reconciler is idempotent and retry-safe',
                      'Human-gated failover by default',
                    ].map((item) => (
                      <li key={item} className="flex items-start gap-3">
                        <CheckCircle2 className="w-4 h-4 text-[var(--teal)] shrink-0 mt-0.5" />
                        {item}
                      </li>
                    ))}
                  </ul>
                </div>
                <div className="relative">
                  <div className="absolute inset-0 bg-gradient-to-r from-[var(--accent)] to-[var(--teal)] rounded-3xl blur-2xl opacity-20 animate-pulse" />
                  <div className="relative rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-6 shadow-2xl">
                    <div className="flex items-center justify-between mb-6">
                      <div className="flex items-center gap-2">
                        <span className="w-3 h-3 rounded-full bg-[var(--red)]" />
                        <span className="w-3 h-3 rounded-full bg-[var(--amber)]" />
                        <span className="w-3 h-3 rounded-full bg-[var(--green)]" />
                      </div>
                      <span className="text-[var(--muted)] text-xs font-mono">geoctl status</span>
                    </div>
                    <div className="space-y-3 font-mono text-sm">
                      {[
                        { label: 'primary', status: 'online', detail: 'wal-lag 0.2s' },
                        { label: 'secondary-fra', status: 'online', detail: 'receiving' },
                        { label: 'git-rsync', status: 'synced', detail: '12m ago' },
                        { label: 's3-objects', status: 'verified', detail: 'count/size parity' },
                        { label: 'registry', status: 'skipped', detail: '401 — no auth' },
                      ].map((row) => (
                        <div key={row.label} className="flex items-center justify-between p-3 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)]">
                          <span className="text-[var(--text)]">{row.label}</span>
                          <div className="flex items-center gap-3">
                            <span className={clsx(
                              'text-xs font-bold px-2 py-0.5 rounded uppercase',
                              row.status === 'online' || row.status === 'synced' || row.status === 'verified'
                                ? 'bg-[var(--green)]/10 text-[var(--green)]'
                                : row.status === 'skipped'
                                  ? 'bg-[var(--amber)]/10 text-[var(--amber)]'
                                  : 'bg-[var(--red)]/10 text-[var(--red)]'
                            )}>
                              {row.status}
                            </span>
                            <span className="text-[var(--muted)] text-xs hidden sm:inline">{row.detail}</span>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            </Reveal>
          </section>

          {/* Capabilities — tabbed */}
          <section id="capabilities">
            <Reveal>
              <div className="text-center max-w-3xl mx-auto mb-14">
                <SectionTag color="teal">Capabilities</SectionTag>
                <h2 className="mt-4 text-3xl md:text-4xl font-display font-bold">Production-grade geo-replication, end to end.</h2>
                <p className="mt-4 text-[var(--muted)]">
                  Each capability is implemented as an idempotent reconciler that observes state,
                  repairs drift when safe, and reports what it finds.
                </p>
              </div>
            </Reveal>

            <Reveal delay={150}>
              <div className="grid lg:grid-cols-12 gap-6">
                <div className="lg:col-span-4 space-y-3">
                  {capabilities.map((cap) => (
                    <button
                      key={cap.id}
                      onClick={() => setActiveCap(cap.id)}
                      className={clsx(
                        'w-full text-left p-4 rounded-xl border transition-all flex items-start gap-3 group',
                        activeCap === cap.id
                          ? 'bg-[var(--surface-elevated)] border-[var(--accent)] shadow-lg shadow-[var(--accent-glow)]'
                          : 'bg-[var(--surface)]/40 border-[var(--border)] hover:border-[var(--border-strong)]'
                      )}
                    >
                      <div className={clsx(
                        'w-2 h-2 rounded-full mt-2 shrink-0 transition-colors',
                        activeCap === cap.id ? 'bg-[var(--accent)]' : 'bg-[var(--muted)]/40 group-hover:bg-[var(--accent)]/60'
                      )} />
                      <div className="min-w-0">
                        <div className={clsx('font-semibold', activeCap === cap.id ? 'text-[var(--text)]' : 'text-[var(--muted)] group-hover:text-[var(--text)]')}>
                          {cap.title}
                        </div>
                        <div className="text-xs text-[var(--muted)] mt-0.5 truncate">{cap.tagline}</div>
                      </div>
                    </button>
                  ))}
                </div>

                <div className="lg:col-span-8">
                  {capabilities.map((cap) => (
                    <div
                      key={cap.id}
                      className={clsx(
                        'h-full rounded-2xl border p-8 md:p-10 transition-all duration-500',
                        activeCap === cap.id
                          ? 'bg-[var(--surface-elevated)] border-[var(--accent)]/40 opacity-100'
                          : 'hidden'
                      )}
                    >
                      <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-[var(--accent)]/10 text-[var(--accent-light)] text-xs font-bold uppercase tracking-wider border border-[var(--accent)]/20 mb-5">
                        {cap.tagline}
                      </div>
                      <h3 className="text-2xl md:text-3xl font-display font-bold mb-4">{cap.title}</h3>
                      <p className="text-[var(--muted)] leading-relaxed text-lg mb-8">{cap.summary}</p>
                      <ul className="space-y-3">
                        {cap.bullets.map((bullet) => (
                          <li key={bullet} className="flex items-start gap-3 text-sm text-[var(--muted)]">
                            <CheckCircle2 className="w-4 h-4 text-[var(--teal)] shrink-0 mt-0.5" />
                            {bullet}
                          </li>
                        ))}
                      </ul>
                    </div>
                  ))}
                </div>
              </div>
            </Reveal>
          </section>

          {/* Architecture bento */}
          <section id="architecture">
            <Reveal>
              <div className="flex items-center justify-between mb-10">
                <div>
                  <SectionTag color="indigo">
                    <Boxes className="w-3.5 h-3.5" />
                    Architecture
                  </SectionTag>
                  <h2 className="mt-3 text-3xl md:text-4xl font-display font-bold">Infrastructure-level replication.</h2>
                </div>
              </div>
            </Reveal>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
              {architecture.map((card, i) => (
                <Reveal key={card.title} delay={100 + i * 80}>
                  <div
                    className={clsx(
                      'group p-6 rounded-2xl border border-[var(--border)] bg-[var(--surface)]/60 hover:bg-[var(--surface-elevated)] hover:border-[var(--accent)]/30 transition-all',
                      card.span || ''
                    )}
                  >
                    <div className="w-11 h-11 rounded-xl bg-gradient-to-br from-[var(--accent)]/20 to-[var(--teal)]/10 border border-[var(--border)] flex items-center justify-center mb-5 group-hover:border-[var(--accent)]/40 transition-colors">
                      <card.icon className="w-5 h-5 text-[var(--accent-light)]" />
                    </div>
                    <h3 className="font-display font-bold text-xl mb-2">{card.title}</h3>
                    <p className="text-[var(--muted)] text-sm leading-relaxed">{card.body}</p>
                  </div>
                </Reveal>
              ))}
            </div>
          </section>

          {/* CLI commands */}
          <section id="commands">
            <Reveal>
              <div className="text-center max-w-2xl mx-auto mb-14">
                <SectionTag color="amber">
                  <Terminal className="w-3.5 h-3.5" />
                  CLI
                </SectionTag>
                <h2 className="mt-4 text-3xl md:text-4xl font-display font-bold">One binary, one control plane.</h2>
              </div>
            </Reveal>

            <Reveal delay={150}>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {commands.map((c) => (
                  <div key={c.cmd} className="group flex items-center justify-between gap-4 p-4 rounded-xl bg-[var(--surface-elevated)] border border-[var(--border)] hover:border-[var(--accent)]/50 hover:-translate-y-0.5 transition-all">
                    <div className="min-w-0">
                      <div className="font-mono font-semibold text-[var(--text)] text-sm">{c.cmd}</div>
                      <div className="text-xs text-[var(--muted)] mt-0.5">{c.desc}</div>
                    </div>
                    <CopyButton text={c.cmd} />
                  </div>
                ))}
              </div>
            </Reveal>
          </section>

          {/* Quickstart + Deployment + Changelog */}
          <section id="quickstart">
            <Reveal>
              <SectionTag color="teal">
                <HardDrive className="w-3.5 h-3.5" />
                Quickstart
              </SectionTag>
              <h2 className="mt-4 text-3xl md:text-4xl font-display font-bold mb-6">From zero to replica.</h2>
            </Reveal>

            <Reveal delay={100}>
              <div className="relative rounded-2xl border border-[var(--border)] bg-[var(--surface)] overflow-hidden">
                <div className="flex items-center justify-between px-5 py-3 border-b border-[var(--border)] bg-[var(--surface-elevated)]">
                  <span className="text-xs font-mono text-[var(--muted)]">geoctl-quickstart.sh</span>
                  <CopyButton text={quickstart} />
                </div>
                <pre className="p-6 text-sm leading-7 font-mono text-[var(--text)] overflow-x-auto">
                  <code>{quickstart}</code>
                </pre>
              </div>
            </Reveal>

            <div className="grid md:grid-cols-2 gap-6 mt-6">
              <Reveal delay={200}>
                <div className="rounded-2xl border border-[var(--border)] bg-[var(--surface)]/60 p-6 h-full">
                  <div className="flex items-center gap-2 mb-4">
                    <Server className="w-5 h-5 text-[var(--accent)]" />
                    <h3 className="font-display font-bold text-lg">Deployment</h3>
                  </div>
                  <ul className="space-y-4 text-sm">
                    {[
                      { label: 'Binary', value: 'Download release or go install' },
                      { label: 'Config', value: 'geoctl init -o config.yaml' },
                      { label: 'Runtime', value: 'Linux x86_64, Go 1.25+' },
                      { label: 'Upgrade', value: 'Replace binary, restart geoctl serve' },
                    ].map((d) => (
                      <li key={d.label} className="border-b border-[var(--border)] last:border-0 pb-3 last:pb-0">
                        <div className="text-[var(--muted)] text-xs uppercase font-bold tracking-wider">{d.label}</div>
                        <div className="text-[var(--text)] mt-0.5">{d.value}</div>
                      </li>
                    ))}
                  </ul>
                </div>
              </Reveal>

              <Reveal delay={300}>
                <div className="rounded-2xl border border-[var(--border)] bg-[var(--surface)]/60 p-6 h-full">
                  <div className="flex items-center justify-between mb-4">
                    <div className="flex items-center gap-2">
                      <Activity className="w-5 h-5 text-[var(--teal)]" />
                      <h3 className="font-display font-bold text-lg">Latest changes</h3>
                    </div>
                    <a href="https://github.com/lknappich/gitlab-geo-sync/blob/main/CHANGELOG.md" className="text-xs text-[var(--accent-light)] hover:underline">View all</a>
                  </div>
                  <div className="space-y-4">
                    {changelog.map((entry, i) => (
                      <div key={i} className="flex gap-3 text-sm">
                        <span className={clsx(
                          'shrink-0 h-fit px-2 py-0.5 rounded text-[10px] font-bold uppercase',
                          entry.tag === 'Added' ? 'bg-[var(--green)]/10 text-[var(--green)]' : 'bg-[var(--accent)]/10 text-[var(--accent-light)]'
                        )}>
                          {entry.tag}
                        </span>
                        <span className="text-[var(--muted)]">{entry.text}</span>
                      </div>
                    ))}
                  </div>
                </div>
              </Reveal>
            </div>
          </section>

          {/* Documentation */}
          <section id="docs">
            <Reveal>
              <div className="text-center max-w-3xl mx-auto mb-14">
                <SectionTag color="indigo">Documentation</SectionTag>
                <h2 className="mt-4 text-3xl md:text-4xl font-display font-bold">How it works.</h2>
                <p className="mt-4 text-[var(--muted)]">
                  A condensed guide to the architecture. Read the full document in{" "}
                  <a href="https://github.com/lknappich/gitlab-geo-sync/blob/main/docs/architecture.md" className="text-[var(--accent-light)] hover:underline">
                    docs/architecture.md
                  </a>.
                </p>
              </div>
            </Reveal>

            <div className="grid md:grid-cols-2 gap-6">
              {[
                {
                  title: 'Replication strategy',
                  body: 'Replication is infrastructure-level because only that approach can produce a byte-identical replica. PostgreSQL physical streaming, rsync/git fetch for repositories, and cloud-provider or rsync-based object storage work together.',
                },
                {
                  title: 'PostgreSQL WAL streaming',
                  body: 'The secondary is bootstrapped with pg_basebackup, then kept in sync by a WAL receiver. This copies every table, ID, timestamp, and encrypted column without application-level logic.',
                },
                {
                  title: 'Git data',
                  body: 'Two modes are supported: rsync the /var/opt/gitlab/git-data/repositories tree, or run per-project git fetch --prune across all projects discovered in the replicated database.',
                },
                {
                  title: 'Object storage',
                  body: 'For S3, the tool verifies cross-region replication by comparing object count and total size. For disk-backed storage, it rsyncs uploads, artifacts, LFS, packages, and registry blob directories.',
                },
                {
                  title: 'Encrypted columns',
                  body: 'The secondary shares the primary db_key_base so the GitLab application itself can decrypt secrets and tokens. geoctl never decrypts anything; it only copies and compares the key bytes.',
                },
                {
                  title: 'Control plane',
                  body: 'Every component is a reconciler: an idempotent function that observes state and repairs drift when safe. The runner sweeps all reconcilers on a configurable interval and a single failure never aborts the others.',
                },
                {
                  title: 'Failover',
                  body: 'A health-check loop monitors the primary. After a configurable number of consecutive failures it declares the primary down. Promotion is human-gated by default: stop services, promote PostgreSQL, disable read-only, start services, verify db_key_base.',
                },
                {
                  title: 'Clean-room policy',
                  body: 'This is not GitLab Geo. The project contains no GitLab EE code, uses no internal APIs, and does not reimplement Ruby-side crypto. All behavior is derived from public documentation and observable state.',
                },
              ].map((doc, i) => (
                <Reveal key={doc.title} delay={100 + i * 80}>
                  <div className="h-full rounded-2xl border border-[var(--border)] bg-[var(--surface)]/60 p-6 hover:bg-[var(--surface-elevated)] hover:border-[var(--accent)]/30 transition-all">
                    <h3 className="font-display font-bold text-lg mb-2">{doc.title}</h3>
                    <p className="text-[var(--muted)] text-sm leading-relaxed">{doc.body}</p>
                  </div>
                </Reveal>
              ))}
            </div>
          </section>

          {/* Trust / quote */}
          <section>
            <Reveal>
              <div className="relative rounded-3xl overflow-hidden border border-[var(--border)] bg-gradient-to-br from-[var(--surface)] to-[var(--surface-elevated)] p-10 md:p-14">
                <div className="absolute top-0 right-0 w-72 h-72 bg-[var(--accent)]/10 rounded-full blur-3xl" />
                <blockquote className="relative text-xl md:text-2xl font-display font-medium leading-relaxed max-w-3xl">
                  “Geo-replication should be infrastructure-level: WAL streaming, rsync, git fetch, and bucket replication.
                  Not a hidden feature behind an enterprise license.”
                </blockquote>
                <div className="relative mt-6 flex items-center gap-3">
                  <div className="w-10 h-10 rounded-full bg-gradient-to-br from-[var(--accent)] to-[var(--teal)]" />
                  <div>
                    <div className="font-semibold text-[var(--text)]">gitlab-geo-sync maintainers</div>
                    <div className="text-sm text-[var(--muted)]">Clean-room, Apache-2.0 implementation</div>
                  </div>
                </div>
              </div>
            </Reveal>
          </section>
        </main>

        {/* Footer */}
        <footer className="border-t border-[var(--border)] bg-[var(--surface)]">
          <div className="max-w-6xl mx-auto px-6 py-12 flex flex-col md:flex-row items-center justify-between gap-6">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-[var(--accent)] to-[var(--teal)] flex items-center justify-center">
                <Radio className="w-4 h-4 text-white" />
              </div>
              <span className="font-display font-bold text-[var(--text)]">gitlab-geo-sync</span>
            </div>
            <div className="flex items-center gap-6 text-sm text-[var(--muted)]">
              <a href="https://github.com/lknappich/gitlab-geo-sync/blob/main/LICENSE" className="hover:text-[var(--text)] transition-colors">License</a>
              <a href="https://github.com/lknappich/gitlab-geo-sync/blob/main/SECURITY.md" className="hover:text-[var(--text)] transition-colors">Security</a>
              <a href="https://github.com/lknappich/gitlab-geo-sync" className="hover:text-[var(--text)] transition-colors">GitHub</a>
            </div>
            <p className="text-xs text-[var(--muted)]">Not affiliated with GitLab Inc.</p>
          </div>
        </footer>
      </div>
    </div>
  )
}
