# Source2Metal v3.4.1-RC3 — release

- Retains the corrected PGN parser and existing GAME processing.
- Retains exact complete-line overlay merging, including prefixes and transpositions.
- Removes obsolete application-specific input metadata handling from the general overlay route. Unknown tags do not assign a processing role; neutral output omits them.
- Adds an original six-record example and detailed optional-overlay guides in all seven supported languages.
- Corrects outdated line-admission wording in the general manuals and HTML introduction.
- Updates version/build identification, includes source and reproducible tests.

The example is active input when scanned. Remove it from the entire input tree
for production/comparison runs. RAW depth is applied before sequence deduplication.
Windows amd64 executables are cross-compiled; automated runtime regressions
were run on Linux. A subsequent Windows practice run was reported successful
and its supplied reports confirmed the expected processing and final counts.
The release package preserves the tested executables and all Go source bytes.
Only release documentation, packaging and checksums have been updated.
