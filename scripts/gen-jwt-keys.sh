#!/usr/bin/env bash
# Generates an RS256 keypair and prints the JWT_PRIVATE_KEY / JWT_PUBLIC_KEY
# lines for .env, with newlines escaped as \n.
set -euo pipefail

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

openssl genpkey -algorithm RSA -out "$tmpdir/jwt.key" -pkeyopt rsa_keygen_bits:2048 2>/dev/null
openssl rsa -in "$tmpdir/jwt.key" -pubout -out "$tmpdir/jwt.pub" 2>/dev/null

priv="$(awk 'BEGIN{ORS="\\n"} {print}' "$tmpdir/jwt.key" | sed 's/\\n$//')"
pub="$(awk 'BEGIN{ORS="\\n"} {print}' "$tmpdir/jwt.pub" | sed 's/\\n$//')"

echo "JWT_PRIVATE_KEY=\"$priv\""
echo "JWT_PUBLIC_KEY=\"$pub\""
