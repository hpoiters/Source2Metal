# Building the Source2Metal v3.2.0-rc1 review candidate

Use Go 1.24.6. The candidate integrates BIN2PGN v0.1.3 for BOOK RAW only.
Stable v3.0.10 and its published executable are unchanged.

In source/Source2Metal_v3.2.0-rc1, run prepare_syzygy_package.ps1 once.
It downloads and verifies the published SyzygyCheck v2.2.0 user package.
This unchanged package is the only embedded utility; source code is separate.

Run tests on the host before setting the Windows target:
```text
go test ./...
go vet ./...
```

Windows build:
```text
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -buildvcs=false -ldflags="-s -w" -o !Source2Metal_v3.2.0-rc1.exe .
!Source2Metal_v3.2.0-rc1.exe -selftest -no-pause
```

From the repository root, package and check all documentation links:
```text
python tools/package_review.py
```

The review workflow runs on Windows, checks tests and selftest, verifies
SyzygyCheck extraction, then uploads the candidate and checksums as workflow
artifacts. It has read-only repository permissions and no release-publishing
step. No tag, Latest release or existing release is changed.

The 7 manuals and HTML introduction describe five input formats, BIN RAW only,
GAME/BOOK output groups, BOOK exact-line deduplication, and unchanged separate
GAME-METAL/CTG-METAL routes. The Windows practice test on real BIN inputs is
still required before approval. Optional external CTG RAW integration fixtures
can be provided through SOURCE2METAL_CTG_RAW_TEST_DIR when running go test.

BIN2PGN provenance: v0.1.3 source supplied in MakeMem 2026-09-10 10:45.
The kernel was converted into an internal Go package; its CLI was removed.
The application's multiline PGN reader and an adapter around the existing
legal SAN parser merge CTG/BIN lines. CTG statistical results and BIN '*' are
preserved in retained records. No book weights are reinterpreted as game wins.
