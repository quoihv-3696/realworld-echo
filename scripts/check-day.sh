#!/usr/bin/env bash
set -euo pipefail

DAY="${1:-}"
if [[ ! "$DAY" =~ ^[1-6]$ ]]; then
  echo "Cách dùng: scripts/check-day.sh <ngày 1-6>" >&2
  exit 2
fi

DIR="$(cd "$(dirname "$0")/.." && pwd)/tests/hurl"
HOST="${HOST:-http://localhost:8080}"

declare -a BY_DAY=(
  ""
  "auth errors_auth"
  "tags comments"
  "articles pagination profiles feed errors_profiles"
  "favorites errors_articles errors_comments errors_authorization"
  ""
)

FILES=()
for ((d = 1; d <= DAY; d++)); do
  for name in ${BY_DAY[$((d - 1))]}; do
    FILES+=("$DIR/$name.hurl")
  done
done

if [ ${#FILES[@]} -eq 0 ]; then
  echo "Ngày 1 chưa có file hurl chạy được (mọi file đều cần đăng ký tài khoản)."
  echo "Chấm ngày 1 bằng: go test -race ./..."
  exit 0
fi

HURL="${HURL:-hurl}"
if ! command -v "$HURL" >/dev/null 2>&1; then
  echo "Chưa có hurl. Cài bằng: brew install hurl" >&2
  exit 2
fi

echo "Chấm ngày $DAY với ${#FILES[@]} file hurl trên $HOST"
"$HURL" --test --jobs 1 \
  --variable "host=$HOST" \
  --variable "uid=$(date +%s)$$" \
  "${FILES[@]}"
