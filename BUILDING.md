# Building Source2Metal v3.2.1 revision 3

Use Go 1.23.2 and Python 3.11 or newer. The release tag is v3.2.1-r3; normal
user filenames retain v3.2.1. The earlier v3.2.1 and v3.2.1-r2 releases
remain untouched.

1. In source/Source2Metal_v3.2.1 run prepare_syzygy_package.ps1. It fetches and
   verifies the unchanged published SyzygyCheck v2.2.0 utility.
2. Run go test ./... and go vet ./... there.
3. Set GOOS=windows, GOARCH=amd64, CGO_ENABLED=0 and build:
   go build -trimpath -buildvcs=false -ldflags="-s -w" -o ../Source2Metal_console/core.exe .
4. In source/Source2Metal_console run go test ./... and go vet ./..., then:
   go build -trimpath -buildvcs=false -ldflags="-s -w" -o ../Source2Metal_v3.2.1/!Source2Metal_v3.2.1.exe .
5. On Windows, from the repository root run:
   python tools/verify_recovered_core.py
   python tools/package_release.py
   python tools/smoke_release.py release_v3.2.1/Source2Metal_v3.2.1_RELEASE.zip

The console source is the user-approved compact ETA implementation. The core
retains the revision-2 stable application directory and progress instrumentation.
Revision 3 changes only the ChessBase progress-display adapter and adds its
independent rotating activity symbol. Converter internals and output algorithms
remain unchanged. See source/Source2Metal_console/PROVENANCE.md.
The approved compressed kernel in validation is a regression reference only;
it is not used as the release kernel. Both release layers are built from source.

The Windows workflow enforces every test before creating v3.2.1-r3 as Latest.
The publisher verifies all earlier release assets and metadata remain intact.
Source archives are supplied automatically by GitHub from the release tag.
