# Building Source2Metal v3.0.9

The release workflow uses Go 1.26.6, Windows/amd64, `CGO_ENABLED=0` and
`-trimpath`. Source code is not embedded in the executable.

Build from `source/Source2Metal_v3.0.9`:

```text
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go test ./...
go vet ./...
go build -trimpath -buildvcs=false -ldflags="-s -w" -o !Source2Metal_v3.0.9.exe .
!Source2Metal_v3.0.9.exe -selftest
```

`syzygycheck_package.zip` is the only nested utility archive. It contains the
current SyzygyCheck executable and concise user documentation, but no source
archive, MakeMem file, validation bundle or historical version.

The matching SyzygyCheck source and build information are in
`source/SyzygyCheck_v2.0.9_V9F`. Earlier versions remain recoverable through Git
history and earlier releases instead of being duplicated in the current tree.

The GitHub Actions workflow checks the source, utility structure, multilingual
HTML markers, executable names and forbidden obsolete references before it
refreshes the release. Authenticode signing changes binary bytes and therefore
requires a new SHA-256 value.
