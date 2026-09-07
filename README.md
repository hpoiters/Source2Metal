
# Source2Metal

Source2Metal is an open-source Windows toolset for building, analysing and improving computer-chess opening-book material.

The project is aimed at practical computer-chess use, with particular attention to reproducibility, transparent source processing and robust opening-book preparation.

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

The project is being prepared for transparent open-source Windows code signing.

Signing-related changes concern release authentication and distribution only. They are not intended to change chess processing, Syzygy checking, user-interface behaviour or program logic.

A separate code-signing policy will be published in this repository.

## License

Source2Metal contains GPL-covered code and is distributed under the **GNU General Public License version 3 or later**, subject to the notices and licensing information contained in the source code and release package.

See the `LICENSE` file in this repository.

## Project goal

The purpose of Source2Metal is practical experimentation with stronger, more reliable and more varied opening-book material for computer-chess engines, while keeping the processing method inspectable and reproducible.
