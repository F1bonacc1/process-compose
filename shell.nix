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
    in
    with pkgs;
    [
      go
      gopls
      gotools
      gnumake
      gnused
    ];
}
