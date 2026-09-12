# Release policy

This policy is the permanent release gate for SyzygyCheck and the default pattern for related project tools.

## User package

A public user release is a complete ZIP, not a loose executable. Its root contains, in this visible order:

1. the primary executable, prefixed with `!`;
2. `1_READ_FIRST_AL.txt`, with a short all-language explanation and pointer to detailed documentation;
3. `Docs_AL/`, with the all-language guide and supporting user documentation;
4. `Program_AL/`, with every support file named by the executable or a result report.

All internal references must use the exact packaged directory and filename. If a program or report names a support
file, the release gate must verify that the file exists at that exact path inside the ZIP.

## Source

Developer source remains separately available in the GitHub repository. Do not place source inside the user ZIP.
Publish one clearly named `_SOURCE.zip` asset as the recommended download; GitHub's automatically generated source
archives remain present because GitHub adds them to every tagged release.

## Required order

1. Complete the program, command/console text, reports and documents.
2. Build the complete candidate ZIP.
3. Afterwards, perform the full consistency check across code-visible terms, all current user documents,
   translations, versions, filenames, directory names, links and report references.
4. Extract and test the ZIP as a user would receive it.
5. Publish or replace release assets only after every check succeeds.

The consistency check is therefore after implementation and packaging, but always before GitHub publication.
Publication must stop on any failed test, missing item, unexpected root item, source leak or inconsistent reference.
When an existing asset must be replaced, upload and verify the candidate under a temporary name before deleting or
renaming the old asset. Do not use a replacement mode that removes the known-good asset before upload success.
