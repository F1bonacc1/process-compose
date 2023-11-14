# Installation

### Download the Binary
Go to the [releases](https://github.com/F1bonacc1/process-compose/releases/latest), download the package for your OS, and copy the binary to somewhere on your PATH.

### Nix
If you have the Nix package manager installed with Flake support, just run:

```sh
# to use the latest binary release
nix run nixpkgs/master#process-compose -- --help

# or to compile from the latest source
nix run github:F1bonacc1/process-compose -- --help
```

To use process-compose declaratively configured in your project `flake.nix`, checkout [process-compose-flake](https://github.com/Platonic-Systems/process-compose-flake).

### Brew (MacOS and Linux)

```shell
brew install f1bonacc1/tap/process-compose
```