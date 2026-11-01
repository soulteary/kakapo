# Changelog

## v2026.11.02

- Omit arbitrary frontend event payloads and JSON decoder details from logs and error responses. Malformed input still returns HTTP 400; valid request and event response shapes are unchanged.
- Keep provider error bodies, malformed success bodies and low-level transport/read errors out of translation diagnostics. Preserve error kinds and HTTP status; retain `APIError.Detail` for source compatibility without populating or formatting it.
- Preserve committed settings and history when a replacement write fails. Sync a private same-directory temporary file, atomically replace the destination and sync its directory. Disclose post-replacement sync uncertainty; reject symlink, non-regular and read-only destinations.
- Report history deletion failures instead of returning success while data remains. An already absent history file remains a successful clear.
- Decode history and nested results once while retaining legacy key compatibility and replacement-on-success semantics. Linux paired microbenchmarks show reduced decode time and allocations; this is not an end-to-end translation latency claim.
- Add Linux behavior tests across configuration, history, translation, HTTP services, event lifecycle and desktop assembly. Require at least 95% total statement coverage in CI and release reports.
- Run universal macOS packaging checks on Go and frontend changes, and correct the runtime, platform and deployment documentation.

### Upgrade and rollback

Stop the application before replacing it; use a single instance per configuration directory. No data format migration or new persistent volume is required. Existing history, logs, backups and exports are not rewritten: restrict access, handle retained or exported copies under the applicable retention policy, and rotate exposed credentials. Older versions can read the same files, but mixed versions or rollback can resume unsafe error forwarding and false-success history clearing. Existing absence of cross-process storage locking is unchanged.

Atomic replacement requires a writable directory, a filesystem supporting file/directory synchronization and same-directory rename, and space for both file versions. Convert intentional symlinks to private regular files before upgrade; preserve required ACL/attribute or hard-link copies separately. Mount the whole configuration directory rather than individual JSON files. After a process exit, private `.kakapo-*.tmp` files may remain; stop all instances and inspect them before retention/cleanup. A committed replacement with a directory-sync error must be inspected before retrying an append. These persistence changes do not introduce a cross-file or Keychain transaction.
