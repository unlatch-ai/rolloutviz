# RLViz documentation

RLViz is an open-source viewer for agent rollouts. It reads existing trace
files, normalizes them into a small event model, and lets you inspect or compare
what the model, tools, grader, and environment did.

The browser viewer runs entirely in the tab. The CLI handles larger files,
private formats, growing traces, and local adapters. Neither path requires an
account or changes the source trace.

## Start here

- [Open the browser viewer](https://rlviz.dev/) for canonical NDJSON, Inspect AI
  EvalLog JSON, or Verifiers GenerateOutputs JSON up to 32 MiB.
- Read [onboarding](onboarding.html) for installation and the first local trace.
- Check [supported formats](supported-formats.html)
  before writing an adapter.
- Use [adapter authoring](adapter-authoring.html) for a private or unsupported
  trace format.

## Install the CLI

```bash
brew install TheSnakeFang/tap/rlviz
```

The same native binary is available through npm or the verified shell
installer:

```bash
npm install --global rlviz
curl -fsSL https://rlviz.dev/install.sh | sh
```

Then inspect and open a trace:

```bash
rlviz inspect ./path/to/rollout.ndjson
rlviz open ./path/to/rollout.ndjson
```

`inspect` probes the file without launching the viewer. `open` starts a
loopback-only daemon, builds a removable local index, opens the browser, and
returns immediately.

## What is different

- **Existing traces, not an instrumentation SDK.** Point RLViz at stored files
  from a run that already happened.
- **Local by default.** Browser parsing stays in the tab. CLI viewing stays on
  loopback and makes no outbound requests.
- **Source-read-only.** Normalized events retain provenance back to raw records;
  titles, descriptions, layouts, and other presentation state remain local.
- **One rollout or a collection.** Read steps at adjustable fidelity, group
  trials, arrange multiple rollouts as rows, pin detail views, and compare
  trajectories around behavioral anchors.
- **Formats stay outside the viewer.** Built-in decoders and explicit local
  adapters map source data into the same canonical records.

## Boundaries

RLViz does not run agents, execute recorded tools, train models, manage prompts,
or provide hosted monitoring. It is an inspection and comparison tool.
