# Source2Metal v3.4.1-RC3

Windows 64-bit release package. The v3.4.1-RC3 build identifier is retained
to identify the exact Windows-tested executables.

Source2Metal processes GAME sources (PGN, CBH, 2CBH) and BOOK sources
(CTG sets, Polyglot BIN, optional opening-line PGN). The existing GAME
selection and corrected PGN parser are retained. BIN remains RAW-only.

OVERLAY-PGN is optional. Put legal opening-line PGN in that directory beneath
an input root, or use a .overlay.pgn suffix. In RAW + METAL mode, distinct
complete canonical lines are appended to Metal.pgn with result *.
Only exact full-line duplicates are removed. Prefixes of different retained
lengths and different move orders remain. The RAW depth limit still applies.
Overlay does not contribute GAME win/draw/loss statistics or book weights.

The package includes an original, freely reusable VoorbeeldOverlay.pgn and
seven overlay guides: DE, EN, ES, FR, NL, RU and ZH. The example is active
input if included in a scan. Remove it or move it outside the complete input
tree before production or comparison runs. Its isolated test produces five
final records from six input records.

Start with the multilingual HTML guide and OVERLAY-PGN/MANUAL_EN.txt.
Source and reproducible tests are included separately; extract source only
into a separate development directory, outside the production input tree.
See BUILDING.md, LICENSE and THIRD_PARTY_NOTICES.md.

CTG format support remains incomplete: underpromotions and special pawnless
symmetries are limited. CompleteCTG remains false. No engine evaluation or
playing-strength guarantee is implied by legal-move validation.
