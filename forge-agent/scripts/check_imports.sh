#!/usr/bin/env bash
# Enforces ADR-0006: only app/model_client.py may import the anthropic SDK.
set -euo pipefail

violations="$(grep -RIl --include='*.py' -E '^\s*(import|from)\s+anthropic' app tests 2>/dev/null \
  | grep -v '^app/model_client\.py$' || true)"

if [[ -n "$violations" ]]; then
  echo "ADR-0006 violation: 'anthropic' imported outside app/model_client.py:"
  echo "$violations"
  exit 1
fi

echo "ok: no anthropic imports outside app/model_client.py"
