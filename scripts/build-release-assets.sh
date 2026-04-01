#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <tag>" >&2
  exit 1
fi

tag="$1"
version="${tag#v}"
project_name="packer-plugin-crusoe"
api_version="x5.0"
repo_root="$(pwd)"
dist_dir="dist"
tmp_dir="$(mktemp -d)"

trap 'rm -rf "$tmp_dir"' EXIT

mkdir -p "$dist_dir"
rm -f "$dist_dir"/*

platforms=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
  "windows amd64"
  "windows arm64"
)

for platform in "${platforms[@]}"; do
  read -r goos goarch <<<"$platform"

  build_dir="$tmp_dir/${goos}_${goarch}"
  mkdir -p "$build_dir"

  binary_name="${project_name}_v${version}_${api_version}_${goos}_${goarch}"
  if [[ "$goos" == "windows" ]]; then
    binary_name="${binary_name}.exe"
  fi

  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
    go build \
    -trimpath \
    -ldflags "-s -w -X main.Version=$version -X main.VersionPrerelease=" \
    -o "$build_dir/$binary_name" \
    .

  archive_name="${project_name}_v${version}_${api_version}_${goos}_${goarch}.zip"
  (
    cd "$build_dir"
    zip -q "$repo_root/$dist_dir/$archive_name" "$binary_name"
  )
done

(
  cd "$dist_dir"
  sha256sum *.zip > "${project_name}_v${version}_SHA256SUMS"
)
