# Source2Metal

Source2Metal is an open-source Windows toolset for building, analysing and
improving computer-chess opening-book material.

## BIN integration review candidate

This branch prepares v3.2.0-rc1 for Windows practice testing. It is not a
published release. The stable v3.0.10 release remains unchanged.
The review package contains:

- `!Source2Metal_v3.2.0-rc1.exe`
- a short HTML introduction with jump links (English, Russian and Chinese)
- a clearly named documentation folder with full manuals in all seven languages

The leading exclamation mark keeps the executable near the top of a folder in
Windows Explorer.

Source2Metal v3.2.0-rc1 includes the published and practically validated
**SyzygyCheck v2.2.0** as its one unpackable utility. This is the same complete
user package as the standalone SyzygyCheck v2.2.0 release. It contains
`!SyzygyCheck_v2.2.0.exe`, `1_READ_FIRST_AL.txt`, `Docs_AL` and `Program_AL`.
The checker can scan either one selected folder or that folder and all real
subfolders, preserves relative paths in checkpoints and offers a selectable
result-report destination.

Source code is deliberately not duplicated inside the EXE or
`Source2Metal_v3.2.0-rc1_RELEASE.zip`. GitHub supplies the source separately and
automatically through **Source code (zip)** and **Source code (tar.gz)**. A
third, manually made source archive would duplicate those downloads and is
therefore not published. Only the current Source2Metal and SyzygyCheck source
trees are present; older versions remain available through Git history and
earlier releases.

## Source material and third-party content

Source2Metal does not include or distribute opening books, chess databases or
third-party game collections. Users supply their own source material and are
responsible for the rights to process, use and redistribute it and any derived
output.

## Conversions and SyzygyCheck

- CTG to PGN — CTG2PGN
- CBH to PGN — CBH2PGN
- 2CBH to PGN — 2CBH2PGN
- Polyglot BIN to complete legal BOOK RAW lines — BIN2PGN v0.1.3
- PGN processing for Fritz and ChessBase opening-book workflows
- SyzygyCheck for integrity checking of local RTBW and RTBZ files

BIN supports RAW only in this candidate. It does not generate BIN-METAL or
change GAME-METAL/CTG-METAL selection. GAME length and Elo filters do not apply
to BIN. RAW outputs are grouped under GAME Sources and BOOK Sources.
Merged BOOK RAW removes exact complete move-sequence duplicates across CTG and
BIN; it preserves the first record, with CTG processed before BIN. Per-source
files remain available. ALL SOURCES RAW combines GAME RAW and merged BOOK RAW
when both exist, without cross-class deduplication.

SyzygyCheck's verification core is based on the original Syzygy tablebase
verification code by Ronald de Man. Source2Metal and SyzygyCheck work locally;
tablebase files, filenames and results are not uploaded.

## Verification and code signing

The review workflow tests the source, builds a Windows executable and uploads
review artifacts only. It cannot publish a release. The executable is unsigned, so
Windows may display a Microsoft Defender SmartScreen or "Unknown publisher"
warning. See [CODE_SIGNING_POLICY.md](CODE_SIGNING_POLICY.md).

## License

Source2Metal contains GPL-covered code and is distributed under the GNU General
Public License version 3 or later, subject to the third-party notices in this
repository. See [LICENSE](LICENSE) and
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
