# Building Source2Metal v3.3.0

Use Go 1.23.2 and Python 3.12 or newer.

1. In `source/Source2Metal_v3.3.0`, run `prepare_syzygy_package.ps1` to fetch
   and verify the unchanged SyzygyCheck v2.2.0 utility.
2. Run `go test ./...` and `go vet ./...` there.
3. Build the core into the launcher directory:
   `go build -trimpath -buildvcs=false -ldflags="-s -w" -o ../Source2Metal_console/core.exe .`
4. In `source/Source2Metal_console`, run `go test ./...` and `go vet ./...`,
   then build:
   `go build -trimpath -buildvcs=false -ldflags="-s -w" -o ../Source2Metal_v3.3.0/!Source2Metal_v3.3.0.exe .`
5. On Windows, run the packaged self-test, `tools/package_release.py`, and
   `tools/smoke_release.py release_v3.3.0/Source2Metal_v3.3.0_RELEASE.zip`.

The GitHub Actions workflow performs these steps on Windows and publishes tag
`v3.3.0` as Latest only after every check succeeds. GitHub supplies the source
archives automatically from that tag.
