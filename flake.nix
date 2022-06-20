{
  description =
    "Process Compose is like docker-compose, but for orchestrating a suite of processes, not containers.";

  # Nixpkgs / NixOS version to use.
  inputs.nixpkgs.url = "nixpkgs";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        lib = pkgs.lib;
      in rec {
        packages = { process-compose = pkgs.callPackage ./default.nix { }; };
        defaultPackage = packages.process-compose;
        apps.process-compose =
          flake-utils.lib.mkApp { drv = packages.process-compose; };
        defaultApp = apps.process-compose;
      });
}
