{
  pkgs ? import <nixpkgs> { },
}:
pkgs.mkShell {
  packages =
    let
      goMod = builtins.readFile ./go.mod;
      goLine = builtins.elemAt (pkgs.lib.splitString "\n" goMod) 2;
      goLineMapped = builtins.replaceStrings [ " " "." ] [ "_" "_" ] goLine;
      go = pkgs."${goLineMapped}";
      swag2op = pkgs.buildGoModule {
        pname = "swag2op";
        version = "v1.0.1";
        src = pkgs.fetchFromGitHub {
          owner = "zxmfke";
          repo = "swagger2openapi3";
          rev = "17d7e5a8f5e12164d3a455f179638c5208869272";
          sha256 = "sha256-0khXtJ2DB56RLMwPU61K/OQld0w16YxPj89AZ31U3yo=";
        };
        subPackages = [ "cmd/swag2op" ];
        vendorHash = "sha256-y6evAKRDgUChEFwVjTIis1aaMJb8sbvRZwIyHyspy3c=";
        doCheck = false;
      };
    in
    with pkgs;
    [
      go
      gopls
      gotools
      gnumake
      gnused
      swag2op
    ];
}
