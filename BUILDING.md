# Building Source2Metal v3.0.9

## Release build

The Source2Metal v3.0.9 release workflow uses Go **1.26.6**, Windows/amd64,
`CGO_ENABLED=0`, `GOAMD64=v1`, and `-trimpath=true`.

The workflow creates the public source archive, embeds that exact archive,
runs the permanent consistency test, builds the Windows executable, runs its
self-test and publishes SHA-256 information.

## Exact v3.0.9 preparation

`source/Source2Metal_v3.0.9/embedded_source.go` embeds a file named
`embedded_source.zip`. Use the exact archive:

`build_inputs/Source2Metal_v3.0.9_SOURCE.zip`

Copy that archive to:

`source/Source2Metal_v3.0.9/embedded_source.zip`

Then build from the Source2Metal source directory with Go 1.26.6:

```text
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go test ./...
go vet ./...
go build -trimpath -buildvcs=false -ldflags="-s -w" -o Source2Metal_v3.0.9.exe .
Source2Metal_v3.0.9.exe -selftest
```

The resulting executable and source-archive hashes are recorded in
`SHA256_Source2Metal_v3.0.9.txt` in the release.

Authenticode signing necessarily changes the binary bytes and therefore produces a different SHA-256 after signing.

## SyzygyCheck

The exact published SyzygyCheck v2.0.9 V9F program and source packages are
embedded unchanged. Its public source and build instructions are preserved
under `source/SyzygyCheck_v2.0.9_V9F/`.
