# Source2Metal

Source2Metal is an open-source Windows toolset for building, analysing and
improving computer-chess opening-book material.

## Source2Metal v3.2.2

v3.2.2 improves long Polyglot BIN runs. BIN processing now shows a live activity
indicator, the current phase, elapsed time and an explicitly approximate overall
remaining-time estimate. The estimate begins as "calculating" and is adjusted as
more of the reachable opening graph becomes known, so the user can make a better
decision about continuing a long run or stopping it.

Ctrl+C during one BIN source now requests a clean stop at the next safe point.
The original BIN remains read-only and an incomplete temporary RAW PGN is never
promoted to final output. In interactive batch mode the user can then choose to
skip only that source and continue with the remaining sources, take a 10-hour
night-rest pause and automatically retry the same source from the beginning, or
stop the complete batch. The night-rest pause resumes without requiring a key
press; Ctrl+C during the pause stops the batch.

v3.2.2 retains the v3.2.1 PGN resilience fixes: one unusable PGN game does not
end a long batch, GAME RAW verification does not depend on an Event tag, and a
high unusable-game rate can stop only the affected PGN source while later
sources continue. Normal GAME filters such as Elo, length, FEN/SetUp and
duplicates are not counted as source errors.

The release retains BIN2PGN v0.1.3 for Polyglot BIN conversion to BOOK RAW.
Earlier releases remain unchanged. The user package contains:

- `!Source2Metal_v3.2.2.exe`
- a short HTML introduction with jump links (English, Russian and Chinese)
- a clearly named documentation folder with full manuals in all seven languages

The leading exclamation mark keeps the executable near the top of a folder in
Windows Explorer.

Source2Metal v3.2.2 includes the published and practically validated
**SyzygyCheck v2.2.0** as its one unpackable utility. This is the same complete
user package as the standalone SyzygyCheck v2.2.0 release. It contains
`!SyzygyCheck_v2.2.0.exe`, `1_READ_FIRST_AL.txt`, `Docs_AL` and `Program_AL`.
The checker can scan either one selected folder or that folder and all real
subfolders, preserves relative paths in checkpoints and offers a selectable
result-report destination.

Source code is deliberately not duplicated inside the EXE or
`Source2Metal_v3.2.2_RELEASE.zip`. GitHub supplies the source separately and
automatically through **Source code (zip)** and **Source code (tar.gz)**. A
third, manually made source archive would duplicate those downloads and is
therefore not published. Older versions remain available through Git history
and earlier releases.

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

BIN supports RAW only in this release. It does not generate BIN-METAL or
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

The release workflow tests the source, verifies safe BIN cancellation, builds
and runs the Windows executable, checks the PGN resilience regression and the
unpacked user package in all seven languages, and publishes only after these
checks pass on the main branch. The executable is unsigned, so Windows may
display a Microsoft Defender SmartScreen or "Unknown publisher" warning. See
[CODE_SIGNING_POLICY.md](CODE_SIGNING_POLICY.md).

## License

Source2Metal contains GPL-covered code and is distributed under the GNU General
Public License version 3 or later, subject to the third-party notices in this
repository. See [LICENSE](LICENSE) and
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
