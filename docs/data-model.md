# Research data model

## Current contract

Canonical v1alpha1 defines run, case, group, trajectory, event, signal,
artifact, and complete records. Events are deliberately generic: kind, ordered
sequence, parent/branch and alignment hints, input/output/data, source location,
raw record, and metadata.

That envelope supports lossless basic viewing and deterministic alignment, but
free-form metadata is not enough for consistent research UX across formats.

## Semantic vocabulary needed next

Before changing the protocol, gather representative traces from agent harnesses,
environment rollouts, post-training pipelines, and evaluation systems. The next
vocabulary should standardize only concepts supported by multiple real formats.

Candidate concepts:

- message role: system, developer, user, assistant, tool, environment
- prompt layer and stable message identity
- turn, span, tool call, and tool result linkage
- subagent, parent, branch, and delegation relationships
- context lifecycle: compaction, truncation, injection, restore
- retained, dropped, and summarized event/message references
- input, cached, reasoning, and output token accounting plus context capacity
- verifier/judge verdict, score, rubric, provenance, and evidence references
- final answer and termination provenance
- source-native versus adapter-derived versus analyzer-inferred status

## Modeling rules

- Preserve the ordered raw event stream even when the UI groups turns or spans.
- Prefer stable references over nested copies of messages or events.
- Store source-native facts separately from adapter-derived and inferred facts.
- Never infer literal branches from independently sampled trajectories.
- Signals carry scalar or textual assessments; events carry ordered activity;
  artifacts carry addressable payloads; analyzer findings remain removable.
- A verifier result can reference supporting events without duplicating them.
- Context changes must express what the source actually knows. Do not fabricate
  before/after membership from token totals alone.
- Every normalized semantic value should retain raw/source provenance.

## Protocol evolution

The protocol remains pre-stable. A semantic revision requires:

1. examples from multiple real source formats
2. a written mapping and provenance model
3. schema and Go/TypeScript type changes
4. canonical, malformed, and compatibility fixtures
5. decoder, validator, index, API, and UI support
6. adapter and analyzer protocol guidance
7. golden rendering and conformance tests
8. an explicit migration note

Do not add UI-only interpretations of arbitrary metadata keys. If a concept is
important enough for core navigation, it needs a canonical contract or a
validated declarative presentation hint.
