#!/usr/bin/env bash
# List GitHub Actions used by the workflows with their pinned and latest release tags,
# and optionally re-pin them to the commit SHA of a newer release.
#
# Usage: pin-actions.sh [--apply minor|major] [workflow dir]
#   (no flag)      report only
#   --apply minor  move each action to the latest release within its current major
#   --apply major  move each action to the latest release overall
# Every pin is written as `owner/repo@<40-char commit sha> # vX.Y.Z`. Annotated tags are
# dereferenced to the commit they point at, since a tag object SHA is not a valid pin.
set -euo pipefail

mode=""
if [[ ${1:-} == --apply ]]; then
  mode=${2:?--apply needs minor or major}
  [[ $mode == minor || $mode == major ]] || { echo "--apply takes minor or major" >&2; exit 2; }
  shift 2
fi
dir=${1:-"$(git rev-parse --show-toplevel)/.github/workflows"}

declare -A refs # repo -> "tag sha" lines, one git ls-remote per repo

tags_of() { # repo -> "tag commit-sha" per vX.Y.Z tag, sorted by version
  if [[ -z ${refs[$1]+x} ]]; then
    refs[$1]=$(git ls-remote --tags "https://github.com/$1" | awk '
      { sub("refs/tags/", "", $2) }
      $2 ~ /\^\{\}$/ { sub(/\^\{\}$/, "", $2); deref[$2] = $1; next }
      { plain[$2] = $1 }
      END { for (t in plain) if (t ~ /^v?[0-9]+\.[0-9]+\.[0-9]+$/) print t, (t in deref ? deref[t] : plain[t]) }
    ' | sort -V -k1,1)
  fi
  printf '%s\n' "${refs[$1]}"
}

major() { sed -E 's/^v?([0-9]+).*/\1/' <<<"$1"; }

printf '%-28s %-42s %-10s %-10s %-10s %s\n' FILE:LINE ACTION CURRENT IN-MAJOR LATEST NOTE
for f in "$dir"/*.yml "$dir"/*.yaml; do
  [[ -e $f ]] || continue
  grep -nP '^\s*(-\s*)?uses:\s*[^\s.][^\s@]*@' "$f" | while IFS= read -r hit; do
    lineno=${hit%%:*}
    line=${hit#*:}
    action=$(grep -oP 'uses:\s*\K[^\s@]+' <<<"$line")
    ref=$(grep -oP 'uses:\s*[^\s@]+@\K\S+' <<<"$line")
    comment=$(grep -oP '#\s*\Kv?[0-9]+\.[0-9]+\.[0-9]+' <<<"$line" || true)
    [[ $action == docker://* ]] && continue
    repo=$(cut -d/ -f1,2 <<<"$action")
    tags=$(tags_of "$repo")

    current=$comment
    if [[ -z $current ]]; then
      if [[ $ref =~ ^[0-9a-f]{40}$ ]]; then
        current=$(awk -v s="$ref" '$2 == s {t = $1} END {print t}' <<<"$tags")
      else
        current=$ref
      fi
    fi

    latest=$(tail -1 <<<"$tags" | awk '{print $1}')
    in_major=""
    [[ -n $current ]] && in_major=$(awk -v m="$(major "$current")" '
      { t = $1; sub(/^v/, "", t); split(t, p, ".") } p[1] == m { last = $1 } END { print last }' <<<"$tags")

    note=""
    [[ $ref =~ ^[0-9a-f]{40}$ ]] || note="not SHA-pinned"
    [[ -n $current && $(major "$current") != "$(major "$latest")" ]] && note="${note:+$note, }MAJOR available"
    [[ -z $current ]] && note="${note:+$note, }current version unknown"
    printf '%-28s %-42s %-10s %-10s %-10s %s\n' "$(basename "$f"):$lineno" "$action" "${current:-?}" "${in_major:--}" "$latest" "$note"

    [[ -n $mode ]] || continue
    target=$latest
    [[ $mode == minor ]] && target=$in_major
    [[ -n $target ]] || continue
    sha=$(awk -v t="$target" '$1 == t {print $2}' <<<"$tags")
    [[ $sha =~ ^[0-9a-f]{40}$ ]] || { echo "  could not resolve $repo $target" >&2; continue; }
    perl -i -pe "if (\$. == $lineno) { s{(uses:\\s*)\\S+(\\s*#.*)?\$}{\${1}$action\@$sha # $target} }" "$f"
  done
done
