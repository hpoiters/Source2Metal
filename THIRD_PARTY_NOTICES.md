# Third-party notices

Source2Metal contains or distributes open-source material from several upstream projects. Their original copyright and license notices remain applicable.

## CTG-related code

Parts of the CTG processing lineage are based on publicly available CTG implementations, including the DroidFish CtgBook lineage by Peter Österlund. The Source2Metal source package identifies the resulting Source2Metal code as GPL v3 or later.

The CTGExtractor lineage also contains portions adapted from `sshivaji/ctgreader` (Copyright © 2016 Shivkumar Shivaji) under the MIT License. Relevant notices are preserved in the source files.

## CBH-related code

The classic CBH route in CB2PGN Builder is a Go port of Dominik Klein's `cbh2pgn 0.1`, originally under the MIT License. The preserved reference source and license are included in the Source2Metal source tree.

## Syzygy `tbcheck`

SyzygyCheck v2.0.9 V9F uses the original `tbcheck` program as its separate
verification core. It first accepts an already present copy only when its
SHA-256 is the expected value. If no local copy is present, it obtains a
temporary copy from the preserved Source2Metal v3.0.7 reference and validates
that copy by SHA-256 before use. Tablebase files, filenames and results are not
uploaded by this helper mechanism.

Based on the original Syzygy tablebase verification code by Ronald de Man.
`tbcheck` originates from the Syzygy tablebase generator project:

https://github.com/syzygy1/tb

The upstream project states that, apart from separately identified third-party files, the relevant source in `src/` is released under **GNU General Public License version 2 only (GPL-2.0-only)**. The upstream `COPYING.txt` and source repository are authoritative for that component.

`tbcheck.exe` is an upstream open-source binary and is not claimed as original
Source2Metal code. It should not be separately signed using a Source2Metal
project certificate unless the signing provider explicitly permits that
treatment. Any future signing arrangement must explicitly permit unchanged
upstream open-source binaries in the package.

## License scope

The root repository license describes the main Source2Metal project. Third-party components remain under their own stated licenses. Nothing in this repository is intended to relicense an upstream component contrary to its original license terms.
