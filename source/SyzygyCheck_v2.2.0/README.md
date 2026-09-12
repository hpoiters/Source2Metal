# SyzygyCheck v2.2.0

Multilingual Windows integrity checker for Syzygy chess tablebases.

SyzygyCheck uses the original `tbcheck` by Ronald de Man as its verification core and builds a multilingual interface,
thread selection, live progress, checkpoint/resume, diagnostics and reporting around it.

## Highlights

- Choice between the control folder only [default] or the control folder plus all subfolders.
- Parent folders are never scanned; symbolic links are not followed outside the control folder.
- Whole-drive scans skip the Windows volume metadata folders `$RECYCLE.BIN` and `System Volume Information`.
- English, German, Dutch, French, Spanish, Chinese and Russian.
- Automatic logical-processor detection.
- Thread choice: all / half (default) / custom.
- Effective verification throughput and remaining time.
- Checkpoint/resume with preservation of confirmed FAIL results.
- Diagnostic onboard self-test before the real scan.
- End-of-run report destination choice: Desktop (first-run default), control folder or an Explorer-selected folder.
- Reports are collected in a language-aware result subfolder (`CheckResult` in all Western interface languages).
- The last selected report destination becomes the Enter-default for the next run.
- Scope and report menus wrap safely with the resized console, including long saved paths and CJK cell widths.
- Every reported FAIL/ERROR includes the full path from the Windows drive letter.
- Detailed result report and recovery guides.

## User release structure

The downloadable user ZIP is required to contain these four root items:

- `!SyzygyCheck_v2.2.0.exe`
- `1_READ_FIRST_AL.txt`
- `Docs_AL/`
- `Program_AL/`

The package must remain together. FAIL/ERROR reports name an exact recovery path under `Program_AL`. Developer
source is not included inside the user ZIP. The clearly named `SyzygyCheck_v2.2.0_SOURCE.zip` release asset is the
recommended source download; GitHub's automatically generated tag archives remain available too.

## Verification core and credit

Based on the original Syzygy tablebase verification code by Ronald de Man.

Upstream project: https://github.com/syzygy1/tb

The AL layer does not reimplement the checksum logic.

## Windows Terminal note

During a real scan SyzygyCheck makes a best-effort attempt to disable `SC_CLOSE` for a classic Windows console host.
The outer X of Windows Terminal belongs to Windows Terminal itself and can still close the window. Checkpoint/resume
remains the dependable protection against losing already confirmed progress.

## Preserved baseline

The unchanged v2.0.9 V9F release remains available as the first practically validated baseline. Its validation
completed 57/57 files (20.22 GB) with 11 workers in 27 seconds, averaging
777.4 MB/s effective verification throughput, finding exactly the two intentional checksum FAILs and 0 read/process
errors. The onboard diagnostic TEST-FAIL also passed.

See `1_READ_FIRST_AL.txt`, `README_AL.txt`, `README_EN.txt`, `README_NL.txt`, `RELEASE_NOTES_v2.2.0.txt` and the
language-specific guides for details. `RELEASE_POLICY.md` records the mandatory build, post-build consistency and
pre-publication package checks. `RELEASE_CONSISTENCY_CHECK_v2.2.0.txt` records the completed check for this repair.
