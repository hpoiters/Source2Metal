# RC3 validation

Go 1.23.2, Linux amd64. Core and launcher: go test ./..., go vet ./...
and native self-test passed. Windows amd64 core and embedded launcher built.
Windows executables were not executed in this environment.

The parser and GAME implementation files are byte-identical to the input
candidate. The final overlay merge function differs only in calling the
neutral emitter without application-specific metadata. Its full-line key,
existing-output copy, prefix/transposition handling and final record checks
are unchanged. General overlay RAW deduplication now ignores all input tags.

Differential synthetic runs against the input candidate:
- GAME only: 64 final records, final PGN byte-identical.
- GAME plus distributed example: 69 records, final PGN byte-identical.
- Distributed example only: 6 read, 1 initial duplicate, 5 final records,
  final PGN byte-identical.
All intermediate PGNs match after normalizing only version labels and test
input paths. Historical comparison logs are not included in this release package.

The example test verifies the retained prefix and transposition, neutral
results, counts and legal moves. Existing tests cover malformed comments,
SAN normalization, invalid moves beyond the depth cap, exact duplicates,
existing base bytes, no-overlay behavior and unchanged GAME counters.

Seven language guides use the same five-part structure, reporting fields,
example counts and depth-limit caveat. The HTML introduction is generated
from the general manuals. Source, documents, example and binaries are
scanned recursively, including nested ZIPs, for excluded project-specific
terms and data identifiers. No such matches remain in the candidate.
This audit covers this prepared package, not a remote repository or history.

Release preparation update (2026-09-29):
The supplied Windows reports identify v3.4.1-RC3 and report successful completion.
The source inventory, GAME counts, individual overlay contributions and final
record count agree with the expected practice result. Private input names and
practice reports are not included in this release. The final practice PGN was
not independently re-parsed during this release preparation.

All 28 candidate content checksums and the candidate archive checksum passed.
The distributed core executable was found byte-for-byte inside its launcher.
All Go source files and both distributed executables are unchanged. No new
compilation is claimed in this preparation environment. Publication status
is tracked separately from this validation summary.
