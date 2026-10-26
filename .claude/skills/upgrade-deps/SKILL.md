---
name: upgrade-deps
description: Upgrade process-compose's Go version, Go module dependencies, golangci-lint, GitHub Actions pins (release versions and commit SHAs), Nix inputs and vendorHash, plus the docs and generated files that depend on them, then verify with race tests, lint, cross-OS builds and the Nix build. Use when asked to upgrade or bump Go, dependencies, linters, CI actions, nixpkgs or the vendor hash, or to "update everything".
argument-hint: "[all|go|deps|lint|actions|nix] [target Go version, e.g. 1.28]"
---

# Upgrade dependencies and toolchain

Scope: `$ARGUMENTS`. Empty or `all` runs every section. Otherwise run the named sections, plus
Survey and Report, which always run. Regenerate (7) and Verify (8) run whenever go.mod or go.sum
changed; an `actions`-only run verifies with the checks at the end of section 6 instead. A Go
version in the arguments (e.g. `1.28`) replaces "latest" as the target. A Go bump always pulls in
the Nix and golangci-lint sections, because both must support the new version.

Scripts live in `.claude/skills/upgrade-deps/scripts/`. Run everything from the repo root.

## Ground rules

- Run `git status --short` first. If files this run will change are already modified (go.mod,
  go.sum, default.nix, flake.lock for `go`, `deps` and `nix`; `.github/workflows/` for
  `actions`), stop and ask. `vendorHash` couples `go.sum` to `default.nix`, so uncommitted work
  mixed in makes the result impossible to commit in parts. Dirty files elsewhere are fine;
  mention them in the report.
- Do not commit. Finish with the report and proposed commit messages.
- Never downgrade. Never move Go past what the whole chain supports: nixos-unstable must have
  `go_1_N`, and a golangci-lint release binary must be built with go1.N or newer. If a link lags,
  stop at the highest version every link supports and report what blocks the rest.
- Fix code for API changes and deprecations; do not silence them with lint exclusions or
  `//nolint`.

## 1. Survey (read-only)

```bash
.claude/skills/upgrade-deps/scripts/versions.sh
.claude/skills/upgrade-deps/scripts/pin-actions.sh
```

`versions.sh` prints every place the Go version is pinned, what nixpkgs offers, which Go built the
pinned and latest golangci-lint binaries, and direct Go deps with updates. `pin-actions.sh` lists
every workflow action with its current, latest-in-major and latest release. Delete any `bin/`
tool it warns about (`rm -f bin/<tool>`), whatever the scope. Show the user a from → to table of
what will move, with MAJOR rows called out, then continue.

## 2. Go version (`go`)

Target: the version from the arguments, else the minor of the latest Go release. Check the gate
first (rows in `versions.sh`): `nixos-unstable go_1_N` is not `missing`, and some golangci-lint
release is built with go1.N or newer. If either fails, do not bump; report it.

| File | Change |
|------|--------|
| `go.mod` | `go mod edit -go=1.N` (bare MAJOR.MINOR, see gotchas) |
| `.github/workflows/go.yml` | `go-version: 1.N`; quote it when the minor ends in 0 (`"1.30"`), or YAML reads 1.3 |
| `default.nix` | `pkgs.go_1_N` |
| `www/docs/contributing.md` | `Go 1.N+` |

Then look for stragglers of the old version and judge each hit:

```bash
git grep -nwF '1.<old minor>' -- ':!go.sum' ':!flake.lock' ':!schemas' ':!src/docs' ':!*.svg'
```

Then delete the cached tool binaries. The Makefile installs them only when missing (`test -s`),
so ones built by an older Go silently stay and break (see gotchas):

```bash
rm -f bin/golangci-lint bin/swag2op
```

## 3. Go dependencies (`deps`)

```bash
go get -u -t ./...
go mod tidy
```

- For every MAJOR row from `versions.sh` (in practice v0 → v1, which `go get -u` takes without
  asking), read the release notes between the two versions and fix compile errors and API changes.
- Check the `go` line in go.mod afterwards. If `go get` raised it to a patch form (`go 1.27.0`),
  run `go mod edit -go=1.N && go mod tidy`. If tidy raises it again, a dependency requires the
  patch form; make `shell.nix` map MAJOR.MINOR instead of leaving the dev shell broken:
  ```nix
  goVersion = builtins.elemAt (pkgs.lib.splitString " " goLine) 1;
  go = pkgs."go_${pkgs.lib.versions.major goVersion}_${pkgs.lib.versions.minor goVersion}";
  ```
- The repo has no `toolchain` line. If `go get` added one, remove it: `go mod edit -toolchain=none`.

## 4. Nix (`nix`, and after any go.mod/go.sum change)

```bash
nix flake update                          # nixpkgs and flake-utils
make nix-update-hash                      # vendorHash from go.sum (vendors into a temp dir)
make build-nix                            # full build, runs go test (doCheck = true)
go version result/bin/process-compose     # must print go1.N.x
nix develop -c go version                 # shell.nix must resolve the same Go
```

- If the build fails with `hash mismatch ... got: sha256-...`, local `go mod vendor` differed from
  Nix's. Put the `got:` value in `vendorHash` and rebuild.
- Leave the `version` attribute in `default.nix` alone. It is the release version (`make nixver`).
- Debug log lines mentioning `/homeless-shelter` during `make build-nix` are sandbox noise.

## 5. golangci-lint (`lint`, and after a Go bump)

- Choose the newest release whose binary is built with go.mod's Go or newer (`versions.sh`,
  "latest release built with"). CI's `golangci-lint-action` downloads that prebuilt binary, so
  the binary's Go is what counts, not the module's `go` directive.
- Update both pins together: `Makefile` (`golangci-lint@vX.Y.Z`) and `.github/workflows/go.yml`
  (`version: vX.Y.Z`).
- `rm -f bin/golangci-lint && make lint`. Fix findings in code.
- Act on deprecation warnings about `.golangci.yaml` (e.g. a linter replaced by a `_v2` one) by
  applying the suggested configuration, then `./bin/golangci-lint config verify -c .golangci.yaml`.

## 6. GitHub Actions (`actions`)

```bash
.claude/skills/upgrade-deps/scripts/check-inputs.py               # baseline: inputs valid at current pins
.claude/skills/upgrade-deps/scripts/pin-actions.sh --apply minor   # newest release within each current major
```

- For each MAJOR row, read the release notes from the current version to the latest
  (`gh release list -R owner/repo`, `gh release view vX.Y.Z -R owner/repo`). Look for Node
  runtime bumps, removed or renamed inputs, and changed defaults (caching, credential
  persistence). Adjust the step's `with:` inputs to match.
- If every major is safe, `pin-actions.sh --apply major`. Otherwise, after `--apply minor`, edit
  the accepted majors by hand. Resolve the SHA with
  `git ls-remote --tags https://github.com/owner/repo 'vX.Y.Z*'` and take the `^{}` line when
  present: an annotated tag's own SHA is a tag object, not a commit.
- Every `uses:` must be `owner/repo@<40-char commit SHA> # vX.Y.Z`. Re-run `pin-actions.sh` and
  confirm nothing says `not SHA-pinned`.
- Re-run `check-inputs.py`. Every `with:` key must still be declared by the action at the new pin,
  and no required input may be missing. Release notes do not reliably mention removed inputs, and
  GitHub only warns about unknown ones at run time.
- Check the workflows still parse:
  ```bash
  python3 -c 'import sys, yaml; [yaml.safe_load(open(f)) for f in sys.argv[1:]]' .github/workflows/*.yml
  ```
- Run the changed workflows with `act` (Docker), in the background, a few minutes each:
  ```bash
  act push -W .github/workflows/go.yml -j build --matrix os:ubuntu-latest
  act push -W .github/workflows/rust.yml -j build
  act pull_request -W .github/workflows/docs.yml -j deploy   # pull_request only builds the site
  act push -W .github/workflows/nix.yml -j build
  ```
  - Never run `stale.yml`, or `docs.yml` as `push`, under act: they act on the real repository
    (labelling and closing issues, `mkdocs gh-deploy --force`).
  - nix.yml's `Nix build` step always fails under act with `home directory '/homeless-shelter'
    exists` (Nix runs unsandboxed in the container). The pre-upgrade pins fail the same way, so
    only the `Install Nix` and `Nix version` steps count; `make build-nix` covers the build.
  - act runs the local tree in place of this repo's `actions/checkout`, so a checkout bump is
    only exercised for real on GitHub.
  - The docs job's `actions/cache` post step warns that `.cache` does not exist. That predates
    any upgrade.

## 7. Regenerate generated files (always)

Library bumps change generated output: `invopop/jsonschema` the schema, `cobra` the CLI docs, the
API annotations and parser the OpenAPI spec. On an unchanged tree these targets reproduce the
committed files exactly, so any diff here comes from the upgrade.

```bash
make build schema docs swag
git diff --stat schemas www/docs/cli src/docs
```

- Review the diff and keep it when the upgrade explains it. If it doesn't (for example swag2op,
  installed `@latest`, changed its output format), report it and leave the decision to the user.
- `crates/client` embeds `src/docs/swagger.json` and `schemas/process-compose-schema.json`. If
  either changed and `cargo` is installed, run `cd crates/client && cargo build --release
  --all-features`. Without cargo, say the Rust client is unverified (rust.yml only builds
  default features).

## 8. Verify (always; every command must pass)

```bash
go build ./... && go vet ./...
GOOS=windows go vet ./... && GOOS=darwin go vet ./...   # CI matrix OSes, test files included
make compile                                            # release targets, incl. 386 and arm
make testrace                                           # ~1-2 min, run in the background
make lint
make build-nix && go version result/bin/process-compose
nix develop -c go version
```

Optional and slow: `make github-workflows` runs go.yml (ubuntu) and nix.yml locally with `act`
(needs Docker). Its nix.yml half cannot pass under act; see the act notes in section 6.

When a test fails, check whether it already failed before the upgrade before blaming it: add a
worktree at `HEAD` in the scratchpad, run the same test there, then `git worktree remove` it.

## 9. Report

- Table: component, old → new (Go, each direct dep that moved, golangci-lint, each action, nixpkgs).
- Each verification command and its result. Name what could not be verified locally: Windows and
  macOS test runs, the real Actions runners, the Rust client without cargo.
- Anything blocked (nixpkgs or golangci-lint lagging a Go release), skipped (majors not taken), or
  left for the user (unexplained regen churn).
- Proposed commits, not made. `vendorHash` ties go.mod/go.sum to `default.nix`, and `go_1_N` ties
  `default.nix` to `flake.lock`, so those always land together:
  1. `chore: upgrade Go to 1.N and update dependencies`: go.mod, go.sum, default.nix, flake.lock,
     go.yml `go-version`, docs, regenerated files, code fixes.
  2. `chore: bump golangci-lint to vX.Y.Z`: Makefile, go.yml `version`, `.golangci.yaml`, lint
     fixes. Fold into 1 when the Go bump needed the newer linter, or CI lint fails between them.
  3. `ci: update GitHub Actions`: the `uses:` lines and any `with:` changes.

## Gotchas (each one has bitten this repo)

- **Stale vendorHash fails without a hash mismatch.** If go.sum changes but `vendorHash` does
  not, Nix reuses the module set cached under the old hash and fails with
  `inconsistent vendoring ... is marked as explicit in vendor/modules.txt, but not explicitly
  required in go.mod`. Fix: `make nix-update-hash`.
- **The go.mod `go` line must stay `go 1.N`.** `shell.nix` turns line 3 of go.mod into a nixpkgs
  attribute, so `go 1.27.0` becomes `pkgs.go_1_27_0`, which does not exist. `go get -u` writes
  the patch form whenever a dependency requires it.
- **`go 1.N` makes GOTOOLCHAIN=auto download go1.N.0**, not the latest patch. CI (`setup-go`) and
  Nix use the latest patch. That is expected, not a mismatch to fix.
- **Cached tool binaries break after a Go bump.** golangci-lint refuses a module targeting a newer
  Go than the one it was built with (`can't load config: the Go language version (go1.X) used
  to build golangci-lint is lower than the targeted Go version (1.Y)`). swag2op built by an
  older Go cannot parse the new stdlib (`method must have no type parameters` in
  `math/rand/v2`). Both survive because of `test -s` in the Makefile: delete them.
- **Lagging upstreams.** nixpkgs and golangci-lint can trail a Go release by days or weeks. The
  Nix job or CI lint goes red if go.mod moves first.
