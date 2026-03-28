#!/usr/bin/env bash
set -euo pipefail

mkdir -p dist

NAME="thetistunnel"

build() {
    local os=$1 arch=$2 arm=$3 out=$4
    printf "Building %-30s" "dist/${out}"
    if [ "$arm" = "-" ]; then
        CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
            go build -trimpath -ldflags="-s -w" -o "dist/${out}" .
    else
        CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" GOARM="$arm" \
            go build -trimpath -ldflags="-s -w" -o "dist/${out}" .
    fi
    echo "OK"
}

build linux   amd64 -  ${NAME}-linux-amd64
build linux   arm64 -  ${NAME}-linux-arm64
build linux   arm   7  ${NAME}-linux-armhf
build darwin  amd64 -  ${NAME}-darwin-amd64
build darwin  arm64 -  ${NAME}-darwin-arm64
build windows amd64 -  ${NAME}-windows-amd64.exe

echo ""
echo "Binaries in dist/:"
ls -lh dist/
