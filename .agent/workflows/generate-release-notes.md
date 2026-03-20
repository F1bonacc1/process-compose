---
description: this workflow generates release notes for the next process-compose release
---

# Generate Release Notes

Generate release notes using changes shown in the diffs.
Only use diffs from commits from the last `vX.Y.Z` git tag
Succinctly describe actual user-facing changes, not every single commit or detail that was made implementing them.

Only add new items not already listed in the ./docs/release-notes.md.
Place new changes at the top of the file.
Do NOT edit or update existing release notes entries.
Do NOT add duplicate entries for changes that have existing release notes entries.
Do NOT add additional entries for small tweaks to features which are already listed in the existing history.

Pay attention to see if changes are later modified or superseded in the commit logs.
The release notes doc should only reflect the *final* version of changes which have evolved within a version's commit history.
If the release notes doc already describes the final behavior, don't document the changes that led us there.

Bullet each item at the start of the line with `-`.
End each bullet with a period.

If the change was made by someone other than Eugene Berger note it at the end of the bullet point as ", by XXX."
If the change addresses an issue, include the issue number in the bullet point as ", addresses issue #XXX.". The fixed issues are mentioned in the git commit message.
The release caption should be `## [v1.Y.0] - YYYY-MM-DD` where `v1.Y.0` is the new version and `YYYY-MM-DD` is the date of the release.
The `Y` in `v1.Y.0` should be calculated as `last_release_Y + num_new_features` where `num_new_features` is the number of new features in the release.
Release footer should be `---` followed by a blank line.
