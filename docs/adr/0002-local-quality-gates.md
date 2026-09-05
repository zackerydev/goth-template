# Enforce exhaustive local quality gates

## Status

Accepted

## Context

The repository needs reproducible checks for generated templates, Go code,
package architecture, prose, vulnerabilities, secrets, and commit messages
without requiring a remote CI system.

## Decision

mise pins the toolchain and exposes two primary commands:

- `mise run fix` formats handwritten sources, regenerates templ output, and
  tidies the Go module.
- `mise run check` runs all read-only verification, including race-tested and
  shuffled tests, 100% eligible-code coverage, static analysis, architecture,
  generated-file synchronization, build, vulnerability, and secret checks.

Lefthook runs `fix`, stages the complete result, and then runs the canonical
checks plus a staged-diff secret scan before each normal commit. Cocogitto
accepts `feat`, `fix`, `docs`, `test`, `refactor`, and `chore` commit types.
Generated templ Go and the thin executable entry point are excluded from
coverage.

## Consequences

Normal commits receive the full local quality signal, and the same commands are
available to developers and agents. The hook intentionally stages all changes,
so contributors must review the staged result. Git can still bypass local hooks
with `--no-verify`; a project that needs centrally enforced checks should add
remote CI.
