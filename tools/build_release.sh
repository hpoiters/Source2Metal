#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export CGO_ENABLED=0
utility=source/Source2Metal_core/syzygycheck_package.zip
if [ ! -f "$utility" ]; then
  curl --fail --location --retry 3 'https://github.com/hpoiters/SyzygyCheck/releases/download/v2.2.0/SyzygyCheck_v2.2.0_RELEASE.zip' -o "$utility"
fi
echo 'd80f8fd70f81b4f06cbd89547a7e346aa08688853b2ce2df25b30e93ccd6cb2c  source/Source2Metal_core/syzygycheck_package.zip' | sha256sum --check
mkdir -p dist
unformatted=$(find source/Source2Metal_core source/Source2Metal_console -name '*.go' -exec gofmt -l {} +)
if [ -n "$unformatted" ]; then printf '%s\n' "$unformatted"; exit 1; fi
(cd source/Source2Metal_core && go test ./... && go vet ./... && GOOS=windows GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags='-s -w' -o ../Source2Metal_console/core.exe . && go build -trimpath -buildvcs=false -o ../../dist/core-native .)
cp source/Source2Metal_console/core.exe dist/Source2Metal_core.exe
(cd source/Source2Metal_console && go test ./... && go vet ./... && GOOS=windows GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags='-s -w' -o '../../dist/!Source2Metal_v3.4.1.exe' .)
dist/core-native -selftest
