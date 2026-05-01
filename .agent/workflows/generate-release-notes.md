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
ALWAYS check every commit message in the release range for issue references (e.g. `Issue #XXX`, `#XXX`, `addresses #XXX`, `fixes #XXX`, `closes #XXX`) and include them in the corresponding bullet — do not skip this step. If a single bullet covers multiple commits, aggregate all referenced issues for that bullet.
The release caption should be `## [v1.Y.0] - YYYY-MM-DD` where `v1.Y.0` is the new version and `YYYY-MM-DD` is the date of the release.
The `Y` in `v1.Y.0` should be calculated as `last_release_Y + num_new_features` where `num_new_features` is the number of new features in the release.
Release footer should be `---` followed by a blank line.

After drafting the release notes entries, scan the documentation under `./www/docs/` (including any docs changed in this release's diffs) and add relevant Markdown links to the corresponding bullet points so users can jump to the feature's documentation. Only link to docs that actually describe the change; do not invent or guess URLs.

Documentation is published to GitHub Pages at `https://f1bonacc1.github.io/process-compose/`. Map source paths under `www/docs/` to published URLs by stripping the `www/docs/` prefix and the `.md` suffix, then appending a trailing slash. Section anchors are preserved as `#anchor-slug`. Examples:

- `www/docs/tui.md` → `https://f1bonacc1.github.io/process-compose/tui/`
- `www/docs/tui.md#shortcuts-configuration` → `https://f1bonacc1.github.io/process-compose/tui/#shortcuts-configuration`
- `www/docs/mcp-server.md#built-in-control-tools` → `https://f1bonacc1.github.io/process-compose/mcp-server/#built-in-control-tools`
- `www/docs/cli/process-compose_analyze_critical-chain.md` → `https://f1bonacc1.github.io/process-compose/cli/process-compose_analyze_critical-chain/`

Use these published GitHub Pages URLs in the release notes links — not relative paths or raw GitHub blob URLs.
