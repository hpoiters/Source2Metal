SyzygyCheck v2.0.7 - SOURCE CODE
=================================

This package preserves the source of the SyzygyCheck v2.0.7 build that was
practically validated by the user on Windows.

ANCHORED UX BEHAVIOUR
Active progress uses ONE adaptive physical console line. Every update is fitted
to the current console width. If Windows Terminal is resized during a run, the
screen is redrawn cleanly so terminal reflow cannot accumulate historical
progress rows.

Treat this as a regression requirement. Do not replace it with a two-line
vertical-cursor renderer without repeating a real Windows Terminal resize test.

SCAN RULE
Only .rtbw and .rtbz files in exactly the same directory as the executable are
checked. No recursion, parent directories, or subdirectories.

CHECKSUM ENGINE
`tbcheck.exe` is the proven checksum engine and is embedded unchanged. Do not
modify it without separate validation.

BUILD
Requires Go 1.23 or newer.

Windows/amd64:
  set GOOS=windows
  set GOARCH=amd64
  set CGO_ENABLED=0
  go test ./...
  go vet ./...
  go build -trimpath -buildvcs=false -ldflags="-s -w" -o SyzygyCheck_v2.0.7.exe .

RELEASE REGRESSION TEST
- use multiple .rtbw/.rtbz files and at least one large file;
- repeatedly resize Windows Terminal while progress is active;
- verify that only one current progress line remains visible;
- include a deliberately damaged file and verify the final error appears below
  `Done:`/the localized completion line;
- verify the English/Russian/Chinese language escape route.
