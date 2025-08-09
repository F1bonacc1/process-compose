#!/bin/bash

# Exit on error
set -e

# Ensure we're in the project root (where go.mod resides)
if [ ! -f "go.mod" ]; then
  echo "Error: go.mod not found in current directory"
  exit 1
fi

if [ ! -f "default.nix" ]; then
  echo "Error: default.nix not found in current directory"
  exit 1
fi

echo "Vendoring dependencies..."
go mod vendor

echo "Computing new vendorHash..."
# Generate the new hash
NEW_HASH=$(nix-hash --type sha256 --sri vendor)

echo "Updating default.nix with new vendorHash: $NEW_HASH"
# Replace the old hash in default.nix
sed -i "s|vendorHash = \"sha256-[a-zA-Z0-9+/]*=*\"|vendorHash = \"$NEW_HASH\"|" default.nix

rm -rI vendor
echo "Done! vendorHash updated successfully."
