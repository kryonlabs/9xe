{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    go
    unicorn
  ];

  # NixOS library linking for CGo
  NIX_LDFLAGS = pkgs.lib.concatStringsSep " " [
    "-L${pkgs.unicorn}/lib"
  ];

  # CGo compiler flags
  CGO_CFLAGS = "-I${pkgs.unicorn}/include";
  CGO_LDFLAGS = "-L${pkgs.unicorn}/lib -lunicorn";

  shellHook = ''
    export LD_LIBRARY_PATH="${pkgs.unicorn}/lib:$LD_LIBRARY_PATH"
  '';
}
