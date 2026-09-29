# Source2Metal v3.4.1

Final release source. Processing logic is unchanged from the Windows-tested
candidate; version identifiers and release text are updated.
Use Go 1.23.2 or compatible newer Go. No external Go module dependencies.

Run `bash tools/build_release.sh` from the source root on Linux with Go on PATH.
The script checks gofmt, runs tests and vet for core and launcher, builds the
Windows amd64 core and launcher, and runs the native core self-test.
The launcher embeds the freshly built core. The unchanged SyzygyCheck utility
is included with its original notices and validated by the core self-test.
Output: `dist/!Source2Metal_v3.4.1.exe` and `dist/Source2Metal_core.exe`.
This script does not upload, tag, push or publish anything.

Direct PGN is a normal GAME source. CBH/2CBH through CB2PGN remain supported
alternative GAME routes. No general superiority of either route is claimed.
