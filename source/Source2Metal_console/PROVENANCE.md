# Approved console and recovered core

Runtime console source comes from 2026-09-16_07-35_Source2Metal_ETA_COMPACT_TEST.zip.
Only its buildMarker changes. validation/approved_console_hashes.json and a Go
test enforce this comparison; test infrastructure changes do not change runtime.

The approved core SHA-256 is
153b232b403929743facf8ac8f1791627a76dbea46a16425fdd6a67cac664392.
Its compressed reference is retained for Windows regression checks.

The missing core repairs were recovered against the normal v3.2.1 source:
applicationDir and its three call sites, BIN liveness without the obsolete note,
and BOOK RAW merge/verification activity. With the original Go 1.23.2 compiler,
395 functions match the reference after instruction relocation normalization.
Differences are restricted to progress functions and their instrumentation of
mergeBookRAW. All converter internals remain the normal v3.2.1 source.
Build-date metadata is updated to 2026-09-16. Every production release builds
core.exe from the recovered Go source, then embeds it in the approved console.
Native Windows PGN parity and menu tests run before publishing.
