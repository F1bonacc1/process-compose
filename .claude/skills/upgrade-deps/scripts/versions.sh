#!/usr/bin/env bash
# Report pinned vs latest versions of Go, nixpkgs Go, golangci-lint and direct Go deps.
# Read-only: changes nothing in the repo.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

os=$(go env GOHOSTOS)
arch=$(go env GOHOSTARCH)

latest_semver_tag() { # repo -> highest vX.Y.Z tag
  git ls-remote --tags --refs "https://github.com/$1" |
    grep -oP 'refs/tags/\Kv[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -1
}

lint_built_with() { # golangci-lint version -> Go version its release binary was built with
  local v=${1#v} dir="$tmp/gcl-$1"
  mkdir -p "$dir"
  curl -fsSL "https://github.com/golangci/golangci-lint/releases/download/v$v/golangci-lint-$v-$os-$arch.tar.gz" |
    tar xz -C "$dir" --strip-components=1 2>/dev/null &&
    go version "$dir/golangci-lint" | awk '{print $2}' || echo "unknown"
}

row() { printf '%-34s %s\n' "$1" "$2"; }

echo "== Go"
mod_go=$(awk '/^go /{print $2; exit}' go.mod)
latest_go=$(curl -fsSL 'https://go.dev/dl/?mode=json' | jq -r '.[0].version')
latest_minor=$(sed -E 's/^go([0-9]+\.[0-9]+).*/\1/' <<<"$latest_go")
row "go.mod go line" "$mod_go"
row "go.mod toolchain line" "$(awk '/^toolchain /{print $2; exit}' go.mod || true)"
row "go.yml setup-go go-version" "$(grep -oP 'go-version:\s*"?\K[0-9.]+' .github/workflows/go.yml || echo '?')"
row "default.nix pkgs.go_*" "$(grep -oP 'pkgs\.go_\K[0-9_]+' default.nix | tr _ .)"
row "www/docs/contributing.md" "$(grep -oP 'Go \K[0-9.]+(?=\+)' www/docs/contributing.md || echo '?')"
row "latest Go release" "$latest_go"
[[ $mod_go =~ ^[0-9]+\.[0-9]+$ ]] ||
  echo "WARNING: go line '$mod_go' is not bare MAJOR.MINOR; shell.nix maps it to pkgs.go_${mod_go//./_}"

echo
echo "== Nix"
attr="go_${latest_minor//./_}"
row "locked nixpkgs $attr" "$(nix eval --raw --inputs-from . "nixpkgs#$attr.version" 2>/dev/null || echo missing)"
row "nixos-unstable $attr" "$(nix eval --raw "github:NixOS/nixpkgs/nixos-unstable#$attr.version" 2>/dev/null || echo missing)"
row "flake package builds with Go" "$(nix eval --raw ".#packages.$(nix eval --impure --raw --expr builtins.currentSystem).process-compose.go.version" 2>/dev/null || echo '?')"

echo
echo "== golangci-lint"
make_lint=$(grep -oP 'golangci-lint@\Kv[0-9.]+' Makefile)
ci_lint=$(grep -A4 'golangci-lint-action' .github/workflows/go.yml | grep -oP 'version:\s*\Kv[0-9.]+' || echo '?')
latest_lint=$(latest_semver_tag golangci/golangci-lint)
row "Makefile pin" "$make_lint"
row "go.yml action version input" "$ci_lint"
row "pinned release built with" "$(lint_built_with "$make_lint")"
row "latest release" "$latest_lint"
row "latest release built with" "$(lint_built_with "$latest_lint")"
[[ $make_lint == "$ci_lint" ]] || echo "WARNING: Makefile and go.yml golangci-lint pins differ"

echo
echo "== Cached tools in bin/ (the Makefile only installs them when missing)"
for tool in golangci-lint swag2op; do
  [[ -x bin/$tool ]] || continue
  built=$(go version "bin/$tool" | awk '{print $2}')
  row "bin/$tool built with" "$built"
  # Strictly older than go.mod's go line (1.27 sorts before 1.27.0, so go1.27.0 is fine)
  if [[ $(printf '%s\n%s\n' "${built#go}" "$mod_go" | sort -V | head -1) == "${built#go}" && ${built#go} != "$mod_go" ]]; then
    echo "WARNING: bin/$tool predates go $mod_go; delete it so make rebuilds it"
  fi
done

echo
echo "== Direct Go dependencies with updates (path current -> latest)"
go list -m -u -f '{{if and .Update (not .Indirect) (not .Main)}}{{.Path}} {{.Version}} {{.Update.Version}}{{end}}' all |
  awk 'NF==3 {
    split($2, a, "."); split($3, b, ".")
    flag = (a[1] != b[1]) ? "   <-- MAJOR, read the changelog" : ""
    printf "%-50s %s -> %s%s\n", $1, $2, $3, flag
  }'
