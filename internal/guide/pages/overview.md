# Welcome to RLViz

RLViz is an open-source viewer for agent rollouts. It puts model output, tool calls, environment observations, graders, rewards, errors, and source provenance into one local workspace.

The sample checkout cohort around this Guide is the live product. Open its rollouts, move between events, adjust the timeline, and change the layout before installing anything.

## Install the local CLI

Homebrew:

```sh
brew install TheSnakeFang/tap/rlviz
```

npm:

```sh
npm install --global rlviz
```

Verified shell installer:

```sh
curl -fsSL https://rlviz.dev/install.sh | sh
```

Then configure RLViz and open a trace:

```sh
rlviz init
rlviz open ./path/to/trace.ndjson
```

The CLI adds persistent indexing, larger cohorts, private adapters, structured queries for coding agents, and named workspaces that an agent can update after the GUI opens.

## What it does

- Visualizes, navigates, filters, groups, and compares recorded trajectories.
- Keeps source traces read-only. Browser parsing stays in the tab; CLI viewing stays on loopback.
- Does not run agents, replay recorded tools, train models, manage prompts, or provide hosted monitoring.
