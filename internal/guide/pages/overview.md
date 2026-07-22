# What RLViz does

RLViz is an open-source viewer for agent rollouts. It reads traces from runs that already happened and shows the model, tool calls, environment observations, graders, rewards, errors, and source provenance in one local workspace.

Use it when raw JSON is too slow to inspect and a custom notebook would repeat work the viewer already knows how to do.

## Product boundary

- RLViz visualizes, navigates, filters, groups, and compares trajectories.
- It does not run agents, replay recorded tools, train models, manage prompts, or provide hosted monitoring.
- Source traces remain read-only. Browser parsing stays in the tab; CLI viewing stays on loopback.

## Two ways to use it

- **Browser:** open `https://rlviz.dev`, use the synthetic cohort immediately, or select a supported local file. Nothing is installed and trace bytes are not uploaded.
- **Local CLI:** install `rlviz` for larger cohorts, growing files, private adapters, agent-readable queries, and named workspaces that a coding agent can update after the GUI opens.
