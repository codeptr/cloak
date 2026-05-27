#!/usr/bin/env bash

set -eu

go install github.com/mitchellh/gox@latest

mkdir -p release

rm -f ./release/*

if [ -z "${v:-}" ]; then
	echo "Version number cannot be null. Run with v=[version] release.sh"
	exit 1
fi

output="{{.Dir}}-{{.OS}}-{{.Arch}}-$v"
osarch="!darwin/arm !darwin/386"

echo "Compiling:"

os="windows linux"
arch="amd64"
pushd cmd/ck-client
CGO_ENABLED=0 gox -ldflags "-X main.version=${v}" -os="$os" -arch="$arch" -osarch="$osarch" -output="$output"
mv ck-client-* ../../release
popd

os="linux"
arch="amd64"
pushd cmd/ck-server
CGO_ENABLED=0 gox -ldflags "-X main.version=${v}" -os="$os" -arch="$arch" -osarch="$osarch" -output="$output"
mv ck-server-* ../../release
popd

sha256sum release/*