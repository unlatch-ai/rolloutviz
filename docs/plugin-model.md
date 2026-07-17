# Plugin and customization model

## Boundaries

RLViz has three extension layers with different trust and ownership:

| Layer | Owns | Execution |
| --- | --- | --- |
| Adapter | Source detection and canonical mapping | Trusted local process |
| Analyzer | Deterministic derived findings and signals | Trusted local process or built in |
| Presentation configuration | Labels, formats, columns, badges, keymaps, theme tokens | Validated declarative data |

Core RLViz owns navigation, layout, accessibility, performance, source/raw
provenance, and the design system.

## Adapters

Adapters translate a source into canonical records. They may preserve
source-specific information in raw records and metadata, but core navigation
semantics require canonical fields. Adapters do not emit HTML, React components,
CSS, or executable renderer code.

Use `adapter-protocol.md` and `adapter-authoring.md` for the executable contract.

## Analyzers

Analyzers consume a bounded canonical trajectory and produce derived findings
and signals with provenance. Results are cached by analyzer identity and input
digest. Removing an analyzer or its cache never changes source or canonical
records.

Analyzers may identify loops, retries, suspicious state transitions, or
domain-specific verifier patterns. They do not control layout or run recorded
actions. Use `analyzer-protocol.md` for the executable contract.

## Declarative presentation

Safe customization can eventually include:

- field display names and descriptions
- scalar formatting and units
- signal grouping and table columns
- semantic badges and landmark categories
- artifact media-type associations
- inspector sections composed from known safe primitives
- command keymap overrides
- validated design-token theme overrides

Every declaration is schema-validated, has bounded size, and renders through
core components. Unknown declarations fail closed with a useful diagnostic.

## Explicit non-goal: arbitrary viewer code

RLViz does not load arbitrary plugin JavaScript or CSS into the main viewer.
Doing so would undermine local trace safety, visual consistency, accessibility,
performance budgets, and the claim that the base product is battle-tested.

If a future use case requires a fully custom visualization, evaluate a separate
sandboxed document/iframe protocol with an explicit trust boundary. Do not
silently expand adapters or analyzers into UI-code plugins.

## Discovery and trust

Future discovery should use explicit project and user plugin directories,
bounded manifest reads, and deterministic probe ranking. Discovery is not trust.
Executable plugins still require an approved path and digest. Presentation-only
configuration may use a separate schema-validation path because it cannot
execute code.
