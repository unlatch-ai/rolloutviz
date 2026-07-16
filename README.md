# RolloutViz

Visualize and compare agent rollouts.

RolloutViz is a local, open-source viewer for agent trajectories. Point it at a trace, open the viewer, and inspect what the model, tools, grader, and environment did step by step.

The long-term goal is a lightweight workbench for people building agent environments and post-training systems:

- open one trajectory without importing it into a hosted platform
- inspect messages, actions, observations, rewards, grader output, and artifacts
- compare trajectories from the same rollout group
- find the first meaningful behavioral divergence
- extend support for private formats with local adapter plugins
- invoke the viewer directly or through coding agents such as Claude Code, Codex, and Cursor

## Status

RolloutViz has an initial single-trajectory vertical slice. It accepts canonical v1alpha1 NDJSON, validates the stream, starts a loopback-only server, and opens an embedded keyboard-first viewer:

```bash
rlviz open ./path/to/trajectory.jsonl
```

Build it and open the included fixture:

```bash
make web-install
make build
./bin/rlviz open ./fixtures/canonical/linear.ndjson
```

The current `open` and `serve` commands run in the foreground. Background daemon reuse and external adapters are the next implementation milestones.

## Design principles

- **Local first.** No account, upload, instrumentation SDK, or hosted service required.
- **Read existing artifacts.** Adapters translate stored trajectory formats into a small canonical event model.
- **Agent extensible.** Unsupported formats should produce enough structured context for a coding agent to implement and validate an adapter.
- **Lossless inspection.** Normalized events always link back to their raw source records.
- **Fast by default.** Large files are streamed and indexed instead of loaded wholesale.
- **Comparison aware.** Rollout groups and parent relationships exist in the data model before their full UI ships.

## Repository map

- [`docs/product-spec.md`](docs/product-spec.md) defines the user experience and scope.
- [`docs/architecture.md`](docs/architecture.md) defines the initial technical architecture.
- [`docs/adapter-protocol.md`](docs/adapter-protocol.md) defines the external adapter boundary.
- [`docs/implementation-plan.md`](docs/implementation-plan.md) breaks the work into testable milestones.

## Development

The core is written in Go. The local React and TypeScript viewer is compiled and embedded in the release binary.

```bash
make web-install
make check
make build
./bin/rlviz version
```

See [`CONTRIBUTING.md`](CONTRIBUTING.md) before sending a change.

## License

Apache 2.0. See [`LICENSE`](LICENSE).
