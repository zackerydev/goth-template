# Isolate local development by Git worktree

## Status

Accepted

## Context

The primary checkout and linked Git worktrees need to run independently.
Memorable fixed ports are useful in the primary checkout, while linked
worktrees need stable derived ports to avoid collisions. HTML development also
needs to reflect edits without introducing a browser reload proxy or a second
server port.

## Decision

`.config/worktree-env.sh` is the single source of truth for local application
configuration. Both `mise run dev` and `mise run run` source it before starting
the application. `mise run dev` then starts pinned Air; Air rebuilds and
restarts the Go process for Go changes, while the renderer reloads HTML on each
request.

The helper resolves the current and primary worktree paths through Git. The
primary checkout uses application port `8888`. Linked worktrees derive a stable
application port from a hash of their absolute path. Moving a worktree may
therefore change its derived port.

`APP_PORT`, with `PORT` as a fallback, overrides the application port. The
helper validates the selected value as an integer from 1 through 65535 and
checks that it is not already occupied. It exports the application URL and an
instance slug and prints them before startup. `APP_RUN_SMOKE=1` bypasses the
occupancy check and server startup only for task-configuration validation.

`.worktrees/` is ignored by Git and excluded from architecture scans so the
primary checkout never treats linked worktree files as part of its own source
tree.

## Consequences

The primary checkout retains a memorable URL, while linked worktrees can run in
parallel without routine port collisions or manual configuration. A rare hash
collision or a port used by another process fails clearly and can be resolved
with an explicit `APP_PORT` or `PORT` override.

The derived identity is stable for a path rather than a branch name. Local run
tasks depend on Git worktree metadata and on either `nc` or Python for occupancy
checks. There is no proxy port, proxy URL, or browser reload process. New local
services that need ports must extend this helper rather than introducing
unrelated fixed defaults.
