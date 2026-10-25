#!/usr/bin/env sh
# Update resrap-rs everywhere it is used:
#   - codegen/          HTTP snippet generator
#   - app/src/rust-core browser WASM generator (rebuilds the committed pkg/)
#   - both Cargo.lock files
#   - README and docs version references
#
# Usage:
#   ./update-resrap.sh            # newest version on crates.io
#   ./update-resrap.sh 1.0.2      # a specific version
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
CODEGEN="$ROOT/codegen"
WASM="$ROOT/app/src/rust-core"
VERSION="${1:-}"

if ! command -v cargo >/dev/null 2>&1; then
	echo "error: cargo is required" >&2
	exit 1
fi
if ! command -v wasm-pack >/dev/null 2>&1; then
	echo "error: wasm-pack is required to rebuild the browser package" >&2
	exit 1
fi

if [ -z "$VERSION" ]; then
	echo "==> resolving the latest resrap-rs release"
	VERSION=$(
		cargo search resrap-rs --limit 1 |
			sed -n 's/^resrap-rs = "\([^"]*\)".*/\1/p'
	)
	if [ -z "$VERSION" ]; then
		echo "error: could not resolve the latest resrap-rs version" >&2
		exit 1
	fi
fi

case "$VERSION" in
	*[!0-9.]* | '')
		echo "error: invalid version: $VERSION" >&2
		exit 1
		;;
esac

echo "==> resrap-rs $VERSION"

set_cargo_version() {
	file=$1
	tmp="$file.tmp"
	sed "s/^resrap-rs = \"[^\"]*\"/resrap-rs = \"$VERSION\"/" "$file" >"$tmp"
	mv "$tmp" "$file"
}

for manifest in "$CODEGEN/Cargo.toml" "$WASM/Cargo.toml"; do
	echo "==> updating $manifest"
	set_cargo_version "$manifest"
done

for dir in "$CODEGEN" "$WASM"; do
	echo "==> refreshing $(basename "$dir")/Cargo.lock"
	(cd "$dir" && cargo update -p resrap-rs --precise "$VERSION")
done

echo "==> rebuilding the browser WASM package"
(cd "$WASM" && wasm-pack build --target web --out-dir pkg)

echo "==> updating version references in README and docs"
sed -i \
	-e "s/resrap-rs [0-9][0-9]*\\.[0-9][0-9]*\\.[0-9][0-9]*/resrap-rs $VERSION/g" \
	-e "s/resrap-rs\` [0-9][0-9]*\\.[0-9][0-9]*\\.[0-9][0-9]*/resrap-rs\` $VERSION/g" \
	"$ROOT/README.md" \
	"$ROOT/app/src/routes/docs/+page.svelte"

echo "==> running codegen tests"
(cd "$CODEGEN" && cargo test)

echo "==> done: resrap-rs $VERSION"
