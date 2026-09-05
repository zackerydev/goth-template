# Default persistence to SQLite, sqlc, and golang-migrate

## Status

Accepted

## Context

The starter application has no product schema, but applications created from
it should not repeatedly reopen the same persistence-tooling decision. The
common alternatives mix two separate choices: a database engine such as SQLite
or PostgreSQL, and a Go access or migration layer such as `database/sql`, pgx,
sqlx, Goose, sqlc, or golang-migrate.

The baseline should preserve the repository's goals: a small dependency graph,
fast local startup, isolated worktrees, typed queries, explicit migrations, and
real tests without a required container or shared port.

## Decision

SQLite is the default persistence engine when the first persistent feature is
added. Use the standard `database/sql` API with a pure-Go SQLite driver. Use
sqlc to generate typed `database/sql` query code and golang-migrate with its
SQLite and embedded-filesystem drivers to apply versioned migrations.

Goose, sqlx, pgx, PostgreSQL, and an ORM are not baseline dependencies. They may
be introduced only when concrete deployment, concurrency, query, or portability
requirements justify a superseding ADR. In particular, pgx is a PostgreSQL
driver and pool, not an alternative needed for the SQLite baseline.

Persistence sources use this layout:

```text
internal/model/migrations/  numbered golang-migrate up/down SQL
internal/model/queries/     named sqlc queries
internal/model/*_gen.go     committed sqlc output
sqlc.yaml                   SQLite/database-sql generation policy
```

Migration and query SQL are authoritative. Generated Go is committed so normal
builds do not require sqlc, but sqlc installation and explicit generation or
verification tasks are introduced with the first persistence feature. The base
application intentionally has no generic generation task. `internal/model`
owns database-shaped types and adapters. Domain workflows live under
`internal/service/<domain>` and compose generated queries; handlers do not call
sqlc query handles directly.

The process opens one database handle, configures foreign keys and a busy
timeout, applies embedded migrations to that same handle, and keeps it open for
the process lifetime. Using the same handle is required for in-memory SQLite.
The SQLite connection limit remains one unless measured concurrency needs and
tests justify another setting.

Without an explicit `DATABASE_URL`, local startup uses an owner-only database
file under the current worktree's ignored `tmp/` directory. Worktrees therefore
have durable but isolated state. Tests use temporary SQLite files or a dedicated
in-memory handle and exercise real constraints and migrations; they do not
require Docker.

The template does not create a placeholder table, empty model package, or fake
repository. The first feature that needs persistence introduces the initial
migration, queries, generated output, dependencies, and model package together
under this decision.

## Consequences

New applications have a clear persistence default without carrying an unused
schema. Local development and database tests remain fast and independent of
network services. SQL stays explicit while sqlc removes handwritten scan and
mapping boilerplate, and golang-migrate keeps schema history separate from
query generation.

SQLite is not appropriate for every deployment topology. An application that
requires multiple database servers, PostgreSQL-specific behavior, or a shared
high-write workload must document and implement that change deliberately rather
than quietly adding pgx, sqlx, Goose, or parallel persistence paths.
