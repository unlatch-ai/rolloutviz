# Agent instructions

Read `docs/product-spec.md`, `docs/architecture.md`, and the relevant protocol document before changing behavior or public interfaces.

Keep the product local-first and source-read-only. Do not add network calls, telemetry, hosted dependencies, or source mutation without an explicit product decision.

Run `make check` before declaring implementation work complete. Protocol changes must update schemas, fixtures, documentation, and conformance tests together.

Prefer small vertical changes that leave a usable path through the CLI, server, and UI. Avoid speculative abstractions for future hosted or enterprise features.
