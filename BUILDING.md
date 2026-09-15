# Building Source2Metal v3.2.2

Use Go 1.24.6 and Python 3.11 or newer. BIN2PGN v0.1.3 supports BOOK RAW only.
Earlier releases and their published executables are unchanged.

In source/Source2Metal_v3.2.2, run prepare_syzygy_package.ps1 once.
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
go build -trimpath -buildvcs=false -ldflags="-s -w" -o !Source2Metal_v3.2.2.exe .
!Source2Metal_v3.2.2.exe -selftest -no-pause
```

From the repository root, package and check all documentation links:
```text
python tools/package_release_v3.2.2.py
```

The release workflow runs on Windows, checks tests and selftest, verifies
SyzygyCheck extraction, then packages and tests the actual unpacked EXE with
tools/smoke_release_v3.2.2.py. Regression coverage includes seven languages,
paths with spaces and Unicode, duplicate BIN books, GAME-filter exemption,
malformed BIN rejection, saved interactive language selection, PGN resilience,
and safe cancellation of BIN conversion without leaving final or temporary RAW
output. The 10-hour night-rest duration is also locked by a test.

The workflow publishes v3.2.2 as Latest only on main, after all checks succeed.
Existing tags and release assets are never rewritten.

The 7 manuals and HTML introduction describe five input formats, BIN RAW only,
GAME/BOOK output groups, BOOK exact-line deduplication, PGN source resilience,
long-BIN progress/ETA, safe per-source interruption, the 10-hour automatic retry
pause, and unchanged separate GAME-METAL/CTG-METAL routes.

BIN2PGN provenance: v0.1.3 source supplied in MakeMem 2026-09-10 10:45.
The kernel was converted into an internal Go package; its CLI was removed.
The application's multiline PGN reader and an adapter around the existing
legal SAN parser merge CTG/BIN lines. CTG statistical results and BIN '*' are
preserved in retained records. No book weights are reinterpreted as game wins.
