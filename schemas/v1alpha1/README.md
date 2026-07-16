# RolloutViz v1alpha1 contracts

Adapters emit UTF-8 NDJSON: one object matching
`canonical-record.schema.json` per line. The schema validates individual records;
the following stream constraints are also normative:

- IDs are non-empty and unique across the stream.
- A referenced run, case, group, trajectory, event, or parent must have appeared
  earlier in the stream.
- Trajectory parents belong to the same group. Event parents and signal/artifact
  event targets belong to the same trajectory.
- Event `sequence` values are non-negative and strictly increasing within each
  trajectory. They need not be contiguous, so adapters can preserve source order
  without renumbering on append.
- Exactly one `complete` record terminates the stream. Its `records` count is the
  number of preceding records and excludes the completion record itself.
- Artifact paths are untrusted source-relative references. Schema validity never
  grants permission to read a path; the host must enforce its registered-root
  policy.

All contracts are unstable until a non-alpha version is declared. A manifest
selects this version with `api_version: rolloutviz.dev/v1alpha1`.
