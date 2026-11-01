# Changelog

## Unreleased

- Keep provider error bodies, malformed success bodies and low-level transport/read errors out of translation diagnostics. Preserve error kinds and HTTP status; retain `APIError.Detail` for source compatibility without populating or formatting it.
- Report history deletion failures instead of returning success while data remains. An already absent history file remains a successful clear.
- Decode history and nested results once while retaining legacy key compatibility and replacement-on-success semantics. Linux paired microbenchmarks show reduced decode time and allocations; this is not an end-to-end translation latency claim.
- Add Linux behavior tests across configuration, history, translation, HTTP services, event lifecycle and desktop assembly. Require at least 95% total statement coverage in CI and release reports.
- Run universal macOS packaging checks on Go and frontend changes, and correct the runtime, platform and deployment documentation.

### Upgrade and rollback

Stop the application before replacing it; use a single instance per configuration directory. No data format migration or new persistent volume is required. Existing history, logs, backups and exports are not rewritten: restrict access, handle retained or exported copies under the applicable retention policy, and rotate exposed credentials. Older versions can read the same files, but mixed versions or rollback can resume unsafe error forwarding and false-success history clearing. Existing absence of cross-process storage locking is unchanged.
