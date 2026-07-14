# gitlab-geo-sync

`gitlab-geo-sync` is an open-source tool that performs **full one-to-one
geo-replication** between self-hosted GitLab instances — a primary and one
or more secondaries — using only public, documented interfaces and standard
infrastructure tooling. It is **not** GitLab Geo (a paid Premium/Ultimate
feature) and contains no GitLab Enterprise Edition code. This is a
clean-room, independent implementation licensed under Apache-2.0.

> Status: **Phase 0 — Foundations**. The control plane (`geoctl`), config
> schema, logging, and metrics scaffolding are in place. Replication
> reconcilers (Postgres streaming, git data, object storage) are Phase 1.

## Why

GitLab's built-in Geo is a paid feature. If you operate your own GitLab
servers and want a true 1:1 replica for disaster recovery, geographic
read-scaling, or migration, you should be able to build one with the same
standard tools you already use to run Postgres and object storage:
physical WAL streaming, `rsync`/`git fetch`, and bucket replication. This
project orchestrates those tools with observability, idempotent
reconciliation, and a safe failover flow.

## Architecture (B-led hybrid)

The primary replication path is **infrastructure-level** because only it
can achieve a true byte-identical replica:

- **PostgreSQL** — physical streaming replication (`pg_basebackup` +
  WAL receiver). Copies every table, ID, timestamp, and encrypted column.
- **Git data** — either `rsync` of `/var/opt/gitlab/git-data/repositories`
  or per-project `git fetch --prune +refs/*:refs/*`, driven by the
  project list already present in the replicated DB.
- **Object storage** — S3 cross-region replication (or `rsync` for
  disk-backed uploads/artifacts/LFS/packages/registry blobs).
- **Encrypted columns** — the secondary shares the primary's
  `db_key_base` so it can decrypt webhook secrets, access tokens, 2FA
  seeds, etc. You own both servers, so this is legitimate; it is the
  one secret that must be shared across sites.
- **Redis** — the secondary runs its own empty Redis; not replicated
  (cache/queue state is not a source of truth and would risk
  double-execution on promotion).
- **Sidekiq** — not replicated; in-flight jobs at the moment of primary
  failure are the only accepted RPO loss.

The API-driven approach (Strategy A) is supported only as an optional
**observational validator** that diffs counts between sites — never as
the replication path, because it cannot reproduce IDs, timestamps,
encrypted columns, CI history, or registry metadata.

See [`docs/architecture.md`](docs/architecture.md) (planned) for the full
design and the honest limits of each strategy.

## Quickstart (Phase 0)

```sh
# 1. Write a config (see deploy/config.example.yaml). All secrets MUST
#    come from the environment via ${VAR} placeholders.
export PG_REPL_PASSWORD=...
export PG_CTRL_PASSWORD=...
export S3_AK=...
export S3_SK=...

# 2. Validate.
geoctl config-validate -c config.yaml

# 3. Serve metrics + reconcilers (Phase 1 will add the reconcilers).
geoctl serve -c config.yaml
```

## Security model

- All secrets are read from environment variables; **never** hard-coded in
  config files. The config loader rejects any `${VAR}` reference that is
  unset or empty.
- The `db_key_base` is the central trust decision: sharing it across sites
  is what makes the replica able to decrypt encrypted columns. Without it,
  the secondary boots but webhooks/tokens/2FA are non-functional. With it,
  you have a true 1:1 replica. Rotate it via GitLab's documented
  maintenance procedure (re-encrypts affected columns) — out of scope for
  this tool to automate.
- This tool does **not** decrypt anything itself. It copies bytes and
  validates behavior post-promotion.

## Clean-room policy

See [`AGENTS.md`](AGENTS.md). No contributor may have read GitLab EE
source for the Geo feature, and no non-public endpoints or internal
binaries are used. All replication uses documented Postgres schema,
standard WAL/rsync/git protocols, and public APIs only.

## License

Apache-2.0. See [`LICENSE`](LICENSE).