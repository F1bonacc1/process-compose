{
  pkgs ? import <nixpkgs> { },
}:
pkgs.mkShell {
  packages = with pkgs; [
    go
    gopls
    gotools
    gnumake
    gnused
  ];
}
