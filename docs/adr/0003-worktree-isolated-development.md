# Isolate local development by Git worktree

## Status

Accepted

## Context

The application and templ live-reload proxy each need a local port. Fixed ports
are convenient in the primary checkout but prevent linked Git worktrees and
concurrent agents from running independently. Manually assigning ports for every
worktree is error-prone, while ephemeral ports are difficult to remember and do
not provide a stable browser URL across restarts.

Local development should remain self-contained and require no additional
process manager or shared coordination service.

## Decision

`.config/worktree-env.sh` is the single source of truth for local instance
configuration. Both `mise run dev` and `mise run run` source it before starting
the application.

The helper resolves the current and primary worktree paths through Git. The
primary checkout uses application port `8888` and templ proxy port `7331`.
Linked worktrees derive a stable application port from a hash of their absolute
path and use the following port for the proxy. Moving a worktree may therefore
change its derived ports.

`APP_PORT`, with `PORT` as a fallback, overrides the application port.
`PROXY_PORT` overrides the live-reload port. The helper validates that both
values are integers from 1 through 65535, that they differ, and that neither is
already occupied. It exports the selected ports, URLs, and an instance slug and
prints them before startup. `APP_RUN_SMOKE=1` bypasses occupancy checks and
server startup only for task-configuration validation.

`.worktrees/` is ignored by Git and excluded from architecture scans so the
primary checkout never treats linked worktree files as part of its own source
tree.

## Consequences

The primary checkout retains memorable URLs, while linked worktrees can run in
parallel without routine port collisions or manual configuration. A rare hash
collision or a port used by another process fails clearly and can be resolved
with explicit overrides.

The derived identity is stable for a path rather than a branch name. Local run
tasks depend on Git worktree metadata and on either `nc` or Python for occupancy
checks. New local services that need ports must extend this helper rather than
introducing unrelated fixed defaults.
