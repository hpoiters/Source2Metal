
# Source2Metal

Source2Metal is an open-source Windows toolset for building, analysing and improving computer-chess opening-book material.

The project is aimed at practical computer-chess use, with particular attention to reproducibility, transparent source processing and robust opening-book preparation.

## Conversions and utilities

Source2Metal provides practical computer-chess conversion and checking tools, including:

- CTG to PGN — CTG2PGN
- CBH to PGN — CBH2PGN
- 2CBH to PGN — 2CBH2PGN
- PGN processing for Fritz and ChessBase opening-book workflows
- SyzygyCheck for Syzygy tablebase validity and integrity checking of RTBW and RTBZ files

## Included tools

### Source2Metal

Source2Metal processes supported chess sources and produces material intended for opening-book construction and learning workflows.

### SyzygyCheck

SyzygyCheck is a separate utility included with Source2Metal releases. It checks local Syzygy tablebase files and reports damaged or invalid files.

## Current release

The current stable release is:

**Source2Metal v3.0.7**

It includes:

- Source2Metal v3.0.7
- SyzygyCheck v2.0.7
- source-code packages
- SHA-256 integrity information
- multilingual documentation

The published v3.0.7 program logic is treated as the validated reference release. Work related to public distribution and code signing must not alter its functional behaviour.

## Source and reproducibility

Source2Metal is distributed together with source code.

The v3.0.7 Windows executable has been verified against its supplied source and can be reproduced from that source with the documented build preparation.

Release hashes are used to detect accidental or unauthorized modification.

## Code signing

The Windows executable in the current Source2Metal v3.0.7 release is unsigned. Windows may therefore display a Microsoft Defender SmartScreen or "Unknown publisher" warning when the program is downloaded and started.

Source2Metal is published with public source code, reproducible GitHub Actions build verification and SHA-256 integrity information.

The project's code-signing policy is available here:

[CODE_SIGNING_POLICY.md](CODE_SIGNING_POLICY.md)

Trusted Windows code signing may be added to future releases when an appropriate signing route becomes available.


## License

Source2Metal contains GPL-covered code and is distributed under the **GNU General Public License version 3 or later**, subject to the notices and licensing information contained in the source code and release package.

See the `LICENSE` file in this repository.

## Project goal

The purpose of Source2Metal is practical experimentation with stronger, more reliable and more varied opening-book material for computer-chess engines, while keeping the processing method inspectable and reproducible.
