# Source2Metal

Source2Metal is an open-source Windows toolset for building, analysing and
improving computer-chess opening-book material.

## Current release

The current stable release contains only the current user files:

- `!Source2Metal_v3.0.9.exe`
- the three-language HTML guide (English, Russian and Chinese)
- a clearly named documentation folder

The leading exclamation mark keeps the executable near the top of a folder in
Windows Explorer.

Source2Metal v3.0.9 includes the validated **SyzygyCheck v2.0.9 V9F** as an
unpackable utility. Its executable retains the name
`!SyzygyCheck_v2.0.9_V9F.exe`.

Source code is deliberately not duplicated inside the EXE or the end-user ZIP.
It is available separately through GitHub's source download for this release.
Only the current Source2Metal and SyzygyCheck source trees are present there;
older versions remain available through Git history and earlier releases.

## Source material and third-party content

Source2Metal does not include or distribute opening books, chess databases or
third-party game collections. Users supply their own source material and are
responsible for the rights to process, use and redistribute it and any derived
output.

## Conversions and SyzygyCheck

- CTG to PGN — CTG2PGN
- CBH to PGN — CBH2PGN
- 2CBH to PGN — 2CBH2PGN
- PGN processing for Fritz and ChessBase opening-book workflows
- SyzygyCheck for integrity checking of local RTBW and RTBZ files

SyzygyCheck's verification core is based on the original Syzygy tablebase
verification code by Ronald de Man. Source2Metal and SyzygyCheck work locally;
tablebase files, filenames and results are not uploaded.

## Verification and code signing

GitHub Actions tests the public source, builds the Windows executable and
publishes SHA-256 integrity information. The current executable is unsigned, so
Windows may display a Microsoft Defender SmartScreen or "Unknown publisher"
warning. See [CODE_SIGNING_POLICY.md](CODE_SIGNING_POLICY.md).

## License

Source2Metal contains GPL-covered code and is distributed under the GNU General
Public License version 3 or later, subject to the third-party notices in this
repository. See [LICENSE](LICENSE) and
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
