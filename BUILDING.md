# Building Source2Metal v3.0.7

## Validated reference build

The distributed Source2Metal v3.0.7 executable reports Go **1.23.2**, Windows/amd64, `CGO_ENABLED=0`, `GOAMD64=v1`, and `-trimpath=true` in its Go build metadata.

The supplied v3.0.7 source archive was previously validated by rebuilding the Windows executable and comparing SHA-256; the resulting executable matched the distributed reference binary byte-for-byte.

## Exact v3.0.7 preparation

`source/Source2Metal_v3.0.7/embedded_source.go` embeds a file named `embedded_source.zip`.
For the validated v3.0.7 build, use the exact archive:

`build_inputs/Source2Metal_v3.0.7_SOURCE.zip`

Copy that archive to:

`source/Source2Metal_v3.0.7/embedded_source.zip`

Then build from the Source2Metal source directory with Go 1.23.2:

```text
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go test ./...
go vet ./...
go build -trimpath -buildvcs=false -ldflags="-s -w" -o Source2Metal_v3.0.7.exe .
```

The reference SHA-256 of the original unsigned v3.0.7 executable is:

`cbe204f523d779908f3c188560cfd12df174ba41ff9f9a479495b5b71a5f43cf`

Authenticode signing necessarily changes the binary bytes and therefore produces a different SHA-256 after signing.

## SyzygyCheck

The embedded SyzygyCheck v2.0.7 executable also reports Go 1.23.2, Windows/amd64, `CGO_ENABLED=0`, `GOAMD64=v1`, and `-trimpath=true`. Its source and build instructions are preserved under `source/SyzygyCheck_v2.0.7/`.
