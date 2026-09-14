# Code signing policy

## Status

Source2Metal is currently unsigned. An open-source signing application to
SignPath Foundation was not approved because the project did not yet meet its
public-visibility requirement. This policy is retained for a possible later
application; it does not imply current approval or certification.
The Source2Metal v3.2.1 executable is unsigned.
No file should be described as SignPath-signed unless its Authenticode signature has actually been verified.

For any future release signed after explicit project approval:

**Free code signing provided by [SignPath.io](https://signpath.io/), certificate by [SignPath Foundation](https://signpath.org/).**

## Team roles

Source2Metal is currently maintained by the GitHub repository owner.

- Committer and reviewer: [hpoiters](https://github.com/hpoiters)
- Approver for signing requests: [hpoiters](https://github.com/hpoiters)

Changes from other contributors must be reviewed before merging. Each code-signing request must be manually approved by the approver.

## Release integrity

Release binaries must be produced from the public source repository by a documented, repeatable build process. Release hashes are published using SHA-256. Signing is an authentication step and must not be used to introduce unreviewed functional changes.

## Privacy

Source2Metal and SyzygyCheck operate on local files. They do not transfer information to other networked systems unless specifically requested by the user or the person operating the software.

See [PRIVACY.md](PRIVACY.md).

## Third-party components

Upstream open-source components retain their own licenses and notices. They must not be signed as if they were authored by the Source2Metal project. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
