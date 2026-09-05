# Enforce exhaustive local quality gates

## Status

Accepted

## Context

The repository needs reproducible checks for Go code, HTML templates, package
architecture, prose, vulnerabilities, secrets, and commit messages without
requiring a remote CI system.

## Decision

mise pins the toolchain and exposes two primary commands:

- `mise run fix` formats HTML, Markdown, and handwritten Go, then tidies the Go
  module. It does not generate template code.
- `mise run check` runs all read-only verification, including race-tested and
  shuffled tests, 100% eligible-code coverage, static analysis, architecture,
  prose, build, vulnerability, and secret checks.

Lefthook runs `fix`, stages the complete result, and then runs the canonical
checks plus a staged-diff secret scan before each normal commit. Cocogitto
accepts `feat`, `fix`, `docs`, `test`, `refactor`, and `chore` commit types.

The coverage policy keeps the 100% file, package, and total thresholds for
eligible handwritten code. The thin executable entry point under `cmd/app` is
excluded because process orchestration is exercised through startup and route
smoke tests rather than line coverage.

## Consequences

Normal commits receive the full local quality signal, and the same commands are
available to developers and agents. There is no generated-template
synchronization gate or template generator to install. If a future persistence
feature adds sqlc or another generator, its pinned tool and explicit generation
or verification task must be introduced with that feature rather than added to
this base preemptively.

The hook intentionally stages all changes, so contributors must review the
staged result. Git can still bypass local hooks with `--no-verify`; a project
that needs centrally enforced checks should add remote CI.
