# Source2Metal v3.4.1 validation

The final release retains the Windows-tested processing logic. Exactly six
Go files differ from the candidate: version constants, launcher build marker,
source-archive display name, the version consistency assertion and release
status text in seven languages, plus the generated source-information report. Parser, overlay and GAME algorithms and all
Go tests are byte-identical to the tested candidate.

Release gates run go test ./..., go vet ./..., gofmt checks, a native self-test,
Windows amd64 builds and build-information checks in all seven languages.
The launcher must embed the exact newly built core. Expected source hashes
protect the reviewed final source. The seven general manuals must match the
HTML guide exactly, and all overlay guides must agree on example counts.

The package contains source, licenses, neutral documentation, the original
six-record demonstration and SHA-256 checksums. Private practice reports and
input files are excluded. Uploaded assets are downloaded and compared with
the verified local files before the release is marked Latest.

The earlier Windows practice reports confirmed successful processing using
the retained logic. The final version-label build has not separately undergone
that full Windows practice run. Changing the labels changes executable bytes;
use the new checksums for this final release.
