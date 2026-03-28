#!/usr/bin/env bash
# Build script for 9xe with optional SDL2 graphics support

echo "Building 9xe..."

# Build inside nix-shell where SDL2 and pkg-config are available
# Use nix-shell to run the build script with proper environment
nix-shell --run '
# Check for SDL2 development libraries inside nix-shell
SDL2_CFLAGS=""
SDL2_LDFLAGS=""
HAS_SDL2="no"

if command -v pkg-config &> /dev/null; then
    if pkg-config --exists sdl2 2>/dev/null; then
        echo "✓ SDL2 detected via pkg-config"
        SDL2_CFLAGS=$(pkg-config --cflags sdl2)
        SDL2_LDFLAGS=$(pkg-config --libs sdl2)
        HAS_SDL2="yes"
        echo "  Building with SDL2 graphics support"
    else
        echo "ℹ SDL2 not found via pkg-config"
        echo "  Building without SDL2 (graphics disabled)"
    fi
else
    echo "ℹ pkg-config not available, skipping SDL2 detection"
    echo "  Building without SDL2 (graphics disabled)"
fi

# Set build tags based on SDL2 availability
if [ "$HAS_SDL2" = "yes" ]; then
    BUILD_TAGS="-tags=sdl2"
else
    BUILD_TAGS=""
fi

# Build with appropriate tags
CGO_CFLAGS="$SDL2_CFLAGS" CGO_LDFLAGS="$SDL2_LDFLAGS" go build -buildvcs=false $BUILD_TAGS -o 9xe ./cmd/9xe

if [ $? -eq 0 ]; then
    echo "✓ Build successful: ./9xe"
    ls -lh 9xe
    if [ "$HAS_SDL2" = "yes" ]; then
        echo "  SDL2 graphics support enabled"
    else
        echo "  Text-only mode (SDL2 not available)"
    fi
else
    echo "✗ Build failed"
    exit 1
fi
'

exit $?
