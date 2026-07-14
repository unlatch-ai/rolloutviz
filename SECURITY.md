# Security policy

RolloutViz reads traces that may contain private code, prompts, credentials, customer data, and local filesystem paths.

Do not report suspected vulnerabilities through a public issue. Use GitHub's private vulnerability reporting for this repository.

## Security expectations

- The viewer binds to loopback by default.
- Source traces are read-only.
- Normal viewing performs no outbound network requests.
- Recorded commands and tools are never re-executed.
- External plugins require explicit trust.
- Trace-provided file paths do not automatically grant filesystem access.

These are product invariants. Changes affecting them require explicit review and security tests.
