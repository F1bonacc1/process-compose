{
  description =
    "Process Compose is like docker-compose, but for orchestrating a suite of processes, not containers.";

  # Nixpkgs / NixOS version to use.
  inputs.nixpkgs.url = "nixpkgs";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ self.overlays."${system}".default ];
        };
      in {
        overlays.default = final: prev: {
          process-compose = final.callPackage ./default.nix {
            #version = self.shortRev or "dirty";
            date = self.lastModifiedDate;
            commit = self.shortRev or "dirty";
          };
        };
        overlay = self.overlays.default;
        packages = { inherit (pkgs) process-compose; };
        defaultPackage = self.packages."${system}".process-compose;
        apps.process-compose = flake-utils.lib.mkApp {
          drv = self.packages."${system}".process-compose;
        };
        apps.default = self.apps."${system}".process-compose;
      });
}
