# Installation

## Download the Binary

### Binary
Go to the [releases](https://github.com/F1bonacc1/process-compose/releases/latest), download the package for your OS, and add to your `$PATH`.

### Install Script
Install script which is very useful in scenarios like CI. Many thanks to GoDownloader for enabling the easy generation of this script.

By default, it installs on the `./bin` directory relative to the working directory:

```shell
sh -c "$(curl --location https://raw.githubusercontent.com/F1bonacc1/process-compose/main/scripts/get-pc.sh)" -- -d
```

It is possible to override the installation directory with the `-b` parameter. On Linux, common choices are `~/.local/bin` and `~/bin` to install for the current user or `/usr/local/bin` to install for all users:

```shell
sh -c "$(curl --location https://raw.githubusercontent.com/F1bonacc1/process-compose/main/scripts/get-pc.sh)" -- -d -b ~/.local/bin
```
!!! warning "Caution"
    On macOS and Windows, `~/.local/bin` and `~/bin` are not added to `$PATH` by default.

## Nix
If you have the Nix package manager installed with Flake support, just run:

```sh
# to use the latest binary release
nix run nixpkgs/master#process-compose -- --help

# or to compile from the latest source
nix run github:F1bonacc1/process-compose -- --help
```

To use process-compose declaratively configured in your project `flake.nix`, checkout [process-compose-flake](https://github.com/Platonic-Systems/process-compose-flake).

## Brew (MacOS and Linux)

```shell
brew install f1bonacc1/tap/process-compose
```
