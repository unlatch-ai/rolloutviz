# ADR 0003: Per-user loopback daemon

- Status: Accepted
- Date: 2026-07-16

## Context

Coding agents need `rlviz open` to return promptly, while researchers need repeated opens to reuse a stable local viewer.

## Decision

The finished local workflow uses one per-user daemon bound to `127.0.0.1`. `rlviz open` starts it when absent, registers the requested source through an authenticated mutation endpoint, opens or focuses a browser URL, prints a machine-readable result when requested, and exits.

Daemon metadata and its secret are stored in a user-only runtime directory. The HTTP listener uses loopback only by default. Development retains `rlviz serve --foreground` as an explicit foreground mode.

Milestone 1 may ship the foreground server before detachment is implemented, but the CLI output and API boundaries must not imply that foreground behavior is permanent.

## Consequences

- Agent shell calls do not remain occupied after daemon support lands.
- Multiple repositories can share one local viewer without sharing source ownership or trust decisions.
- Mutation endpoints require the daemon secret; ordinary local web pages cannot register arbitrary paths.
- Lifecycle commands include `status`, `stop`, and `doctor`, with stale metadata recovery.
