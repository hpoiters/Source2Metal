# SyzygyCheck v2.0.9 V9F

Multilingual Windows integrity checker for Syzygy chess tablebases.

SyzygyCheck uses Ronald de Man's original `tbcheck` as its verification core and builds a multilingual interface,
thread selection, live progress, checkpoint/resume, diagnostics and reporting around it.

## Highlights

- Checks `.rtbw` and `.rtbz` files in the EXE folder.
- English, German, Dutch, French, Spanish, Chinese and Russian.
- Automatic logical-processor detection.
- Thread choice: all / half (default) / custom.
- Effective verification throughput and remaining time.
- Checkpoint/resume with preservation of confirmed FAIL results.
- Diagnostic onboard self-test before the real scan.
- Detailed result report and recovery guides.

## Verification core and credit

Based on the original Syzygy tablebase verification code by Ronald de Man.

Upstream project: https://github.com/syzygy1/tb

The AL layer does not reimplement the checksum logic.

## Windows Terminal note

During a real scan SyzygyCheck makes a best-effort attempt to disable `SC_CLOSE` for a classic Windows console host.
The outer X of Windows Terminal belongs to Windows Terminal itself and can still close the window. Checkpoint/resume
remains the dependable protection against losing already confirmed progress.

## Final validation

The V9F practical validation completed 57/57 files (20.22 GB) with 11 workers in 27 seconds, averaging
777.4 MB/s effective verification throughput, finding exactly the two intentional checksum FAILs and 0 read/process
errors. The onboard diagnostic TEST-FAIL also passed.

See `README_EN.txt`, `README_NL.txt`, `RELEASE_NOTES_v2.0.9.txt` and the language-specific guides for details.
