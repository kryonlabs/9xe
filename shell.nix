{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    go
    unicorn
    pkg-config
    SDL2
    SDL2.dev
    xorg.libX11
    xorg.libXext
    xorg.libXcursor
    xorg.libXinerama
    xorg.libXi
    xorg.libXrandr
    xorg.libXScrnSaver
    xorg.libXxf86vm
  ];

  # NixOS library linking for CGo
  NIX_LDFLAGS = pkgs.lib.concatStringsSep " " [
    "-L${pkgs.unicorn}/lib"
    "-L${pkgs.SDL2}/lib"
  ];

  # CGo compiler flags
  CGO_CFLAGS = "-I${pkgs.unicorn}/include -I${pkgs.SDL2.dev}/include/SDL2 -I${pkgs.xorg.libX11.dev}/include/X11";
  CGO_LDFLAGS = "-L${pkgs.unicorn}/lib -lunicorn -L${pkgs.SDL2}/lib -lSDL2 -L${pkgs.xorg.libX11}/lib -lX11";

  shellHook = ''
    export LD_LIBRARY_PATH="${pkgs.unicorn}/lib:${pkgs.SDL2}/lib:${pkgs.xorg.libX11}/lib:$LD_LIBRARY_PATH"
    export PKG_CONFIG_PATH="${pkgs.SDL2.dev}/lib/pkgconfig:$PKG_CONFIG_PATH"
    echo "✓ SDL2 development environment loaded"
    echo "  - SDL2 libraries: ${pkgs.SDL2}/lib"
    echo "  - SDL2 headers: ${pkgs.SDL2.dev}/include/SDL2"
    pkg-config --modversion sdl2 2>/dev/null && echo "  - SDL2 version: $(pkg-config --modversion sdl2)"
  '';
}
