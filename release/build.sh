#!/usr/bin/env bash
# Builds this branch's provider as the release assets of a tag v0.1.8-fork.N: xaf's v0.1.8
# (ad0e86e) and two fixes, for unknown values after apply (fix/null-unknowns-after-apply)
# and for Delete (fix/delete-uptime-monitor), built with a pinned Go.
#
#   release/build.sh OUT
#
# writes OUT/terraform-provider-hetrixtools_0.1.8_linux_amd64.zip and OUT/SHA256SUMS.
# The build is reproducible: the pinned toolchain alone (GOTOOLCHAIN=local), no cgo, no
# paths or VCS stamp in the binary, modules exactly as go.sum lists them, and a zip with
# fixed timestamps. Run it on the tagged commit and compare SHA256SUMS with the release.
# Linux x86_64; needs curl, sha256sum and python3.
set -euo pipefail

GO_VERSION=1.27.1
GO_SHA256=63d339f0da5ab53635a56f2490a7984dfe12dfcff22ad749f63edaf590168445
VERSION=0.1.8

[ $# -eq 1 ] || { echo "usage: $0 OUT" >&2; exit 2; }
root=$(cd "$(dirname "$0")/.." && pwd)
[ -z "$(git -C "$root" status --porcelain)" ] || { echo "build.sh: the checkout has uncommitted changes" >&2; exit 1; }
mkdir -p "$1"
out=$(cd "$1" && pwd)
work=$(mktemp -d)
trap 'chmod -R u+w "$work"; rm -rf "$work"' EXIT

curl -fsSL -o "$work/go.tar.gz" "https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz"
echo "$GO_SHA256  $work/go.tar.gz" | sha256sum -c --quiet -
tar -xzf "$work/go.tar.gz" -C "$work"

bin=terraform-provider-hetrixtools_v$VERSION
zip=terraform-provider-hetrixtools_${VERSION}_linux_amd64.zip
mkdir -p "$work/tmp"
env GOENV=off GOTOOLCHAIN=local GOFLAGS="-mod=readonly -modcacherw" CGO_ENABLED=0 GOOS=linux \
  GOARCH=amd64 GOAMD64=v1 GOPATH="$work/gopath" GOCACHE="$work/gocache" GOTMPDIR="$work/tmp" \
  "$work/go/bin/go" build -C "$root/terraform-provider" -trimpath -buildvcs=false \
  -ldflags="-s -w -X main.version=$VERSION" -o "$work/$bin" .

# One file, mode 0755, dated 1980-01-01 (zip's earliest), so the zip is the same each time.
python3 - "$work/$bin" "$out/$zip" <<'EOF'
import sys, zipfile
src, dst = sys.argv[1], sys.argv[2]
info = zipfile.ZipInfo(src.rsplit("/", 1)[1], date_time=(1980, 1, 1, 0, 0, 0))
info.external_attr = 0o100755 << 16
info.compress_type = zipfile.ZIP_DEFLATED
with open(src, "rb") as f, zipfile.ZipFile(dst, "w") as z:
    z.writestr(info, f.read(), compresslevel=9)
EOF
(cd "$out" && sha256sum "$zip" >SHA256SUMS && cat SHA256SUMS)
