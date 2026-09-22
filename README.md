# Source2Metal

Source2Metal is an open-source Windows toolset for building, analysing and
improving computer-chess opening-book material.

## Source2Metal v3.3.0 — complete CTG graph extraction

Version 3.3.0 replaces the former CTG route sampling with a complete traversal
of every decoded reachable position/move pair. CTG moves are exported as
neutral PGN lines with result `*`; frequency fields, recommendations and
unverified learning weights are kept out of the PGN.

The CTG conversion now supports workers within one book and a separate CTG
depth setting. A CTG depth of 0 means no preset limit. Each extraction writes a
coverage report and separate JSONL statistics. CTG RAW can be reused directly
for CTG METAL, while GAME and BOOK sources remain separate.

Perfect2023 validation found 56,857 reachable positions, 60,747 unique
position/move pairs, 3,891 transpositions and 15,325 exported lines. All 15,325
PGN records passed legal-move and parser checks. The Windows practical run
completed successfully. Additional route checks confirmed that `1...e5`,
`1...c6` and the Berlin route are present in the PGN.

The user package contains:

- `!Source2Metal_v3.3.0.exe`
- a multilingual HTML introduction with jump links
- a clearly named documentation folder with manuals in seven languages
- release notes, validation summary, privacy and build information

The leading exclamation mark keeps the executable near the top of a folder in
Windows Explorer.

Source2Metal v3.3.0 retains PGN, CBH, 2CBH and Polyglot BIN processing and the
published SyzygyCheck v2.2.0 utility. BIN remains RAW-only. RAW output keeps
GAME and BOOK sources separate, and `METAL/Metal.pgn` remains the combined
neutral training output.

## Known CTG decoder limits

Underpromotions and special pawnless symmetry are not independently verified.
Recognized promotions currently assume a queen, so `CompleteCTG` remains
`false`. Coverage certifies decoded reachable position/move pairs; it does not
claim every possible transposition path or every physical CTG record.

## Source material and third-party content

Source2Metal does not include or distribute opening books, chess databases or
third-party game collections. Users supply their own source material and are
responsible for the rights to process, use and redistribute it and derived
output.

SyzygyCheck's verification core is based on the original Syzygy tablebase
verification code by Ronald de Man. Source2Metal and SyzygyCheck work locally;
source files, tablebase files, filenames and results are not uploaded.

## Verification and code signing

The release workflow tests and vets both Go modules, builds and runs the Windows
executable, runs the packaged self-test, checks the unpacked package in seven
languages, validates all documentation and publishes only after every check
passes. The executable is unsigned, so Windows may display a Microsoft Defender
SmartScreen or "Unknown publisher" warning. See
[CODE_SIGNING_POLICY.md](CODE_SIGNING_POLICY.md).

## License

Source2Metal contains GPL-covered code and is distributed under the GNU General
Public License version 3 or later, subject to the third-party notices in this
repository. See [LICENSE](LICENSE) and
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
