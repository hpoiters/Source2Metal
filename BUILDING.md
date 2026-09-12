# Building Source2Metal v3.0.10

The release workflow uses Go 1.26.6, Windows/amd64, `CGO_ENABLED=0` and
`-trimpath`. Source code is not embedded in the executable.

Build from `source/Source2Metal_v3.0.10`:

```text
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go test ./...
go vet ./...
go build -trimpath -buildvcs=false -ldflags="-s -w" -o !Source2Metal_v3.0.10.exe .
!Source2Metal_v3.0.10.exe -selftest
```

Before testing or building, run `prepare_syzygy_package.ps1` once in the source
directory. It downloads `SyzygyCheck_v2.2.0_RELEASE.zip` and accepts it only
when its SHA-256 is the pinned release value. The downloaded build input is
not stored in Git and is not duplicated in the Source2Metal source archive.
During compilation it becomes the one utility embedded in the Source2Metal EXE.
It contains `!SyzygyCheck_v2.2.0.exe`, `1_READ_FIRST_AL.txt`, `Docs_AL` and
`Program_AL`, but no source archive, MakeMem file, validation bundle or
historical executable.

The matching SyzygyCheck source and build information are in
`source/SyzygyCheck_v2.2.0`. Earlier versions remain recoverable through Git
history and earlier releases instead of being duplicated in the current tree.

The GitHub Actions workflow checks the source, all 22 files in the utility, the
short HTML introduction and its jump links, all seven full manuals, recovery-
guide paths, executable names, fixed published hashes and forbidden obsolete
references before it creates the new release. The workflow uploads only
`Source2Metal_v3.0.10_RELEASE.zip` and its SHA-256 file. GitHub automatically
adds **Source code (zip)** and **Source code (tar.gz)**; no duplicating custom
source ZIP is created. Authenticode signing changes binary bytes and therefore
requires a new SHA-256 value.
