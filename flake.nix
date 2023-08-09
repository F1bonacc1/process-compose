{
  description =
    "Process Compose is like docker-compose, but for orchestrating a suite of processes, not containers.";

  # Nixpkgs / NixOS version to use.
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/release-22.11";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    let
      mkPackage = pkgs: pkgs.callPackage ./default.nix {
        #version = self.shortRev or "dirty";
        date = self.lastModifiedDate;
        commit = self.shortRev or "dirty";
      };
    in
    (flake-utils.lib.eachDefaultSystem (system:
      {
        packages.process-compose = mkPackage nixpkgs.legacyPackages.${system};
        defaultPackage = self.packages."${system}".process-compose;
        apps.process-compose = flake-utils.lib.mkApp {
          drv = self.packages."${system}".process-compose;
        };
        apps.default = self.apps."${system}".process-compose;
      })
    ) // {
      overlays.default = final: prev: {
        process-compose = mkPackage final;
      };
    }
  ;
}
