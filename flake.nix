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
    (flake-utils.lib.eachDefaultSystem (system: let
      pkgs = nixpkgs.legacyPackages.${system};
    in {
        packages.process-compose = mkPackage pkgs;
        defaultPackage = self.packages."${system}".process-compose;
        apps.process-compose = flake-utils.lib.mkApp {
          drv = self.packages."${system}".process-compose;
        };
        apps.default = self.apps."${system}".process-compose;
        checks.default = self.packages."${system}".process-compose.overrideAttrs (prev: {
          doCheck = true;
          nativeBuildInputs = prev.nativeBuildInputs ++ (with pkgs; [python3]);
        });
      })
    ) // {
      overlays.default = final: prev: {
        process-compose = mkPackage final;
      };
    }
  ;
}
