#!/usr/bin/env bash
# Trigger a Coolify deploy for the audiofile app (manual fallback when CI is unavailable).
set -euo pipefail

COOLIFY_BASE_URL="${COOLIFY_BASE_URL:-https://coolify.fergify.work}"
COOLIFY_APP_UUID="${COOLIFY_APP_UUID:-a7enf427mqokx7il22uibvmo}"

if [ -z "${COOLIFY_API_TOKEN:-}" ]; then
	echo "COOLIFY_API_TOKEN is required (Coolify → Keys & Tokens → API tokens)." >&2
	exit 1
fi

curl -fsS -X POST \
	-H "Authorization: Bearer ${COOLIFY_API_TOKEN}" \
	-H "Accept: application/json" \
	"${COOLIFY_BASE_URL%/}/api/v1/deploy?uuid=${COOLIFY_APP_UUID}&force=false"

echo
echo "Deploy queued for app ${COOLIFY_APP_UUID}."
