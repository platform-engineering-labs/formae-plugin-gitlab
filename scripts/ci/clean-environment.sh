#!/bin/bash
# © 2025 Platform Engineering Labs Inc.
# SPDX-License-Identifier: FSL-1.1-ALv2
#
# Clean Environment Hook for GitLab Plugin
# =========================================
# Called before AND after conformance tests to clean up test resources.
# Idempotent — safe to run multiple times.

set -uo pipefail

GROUP="${GITLAB_TEST_GROUP:-platform-engineering-labs1}"
PROJECT="${GITLAB_TEST_PROJECT:-formae-plugin-gitlab-test}"
PROJECT_PATH="${GROUP}/${PROJECT}"
API_URL="https://gitlab.com/api/v4"

# URL-encode the project path
ENCODED_PATH=$(python3 -c "import urllib.parse; print(urllib.parse.quote('${PROJECT_PATH}', safe=''))")

echo "clean-environment.sh: Cleaning GitLab test resources on ${PROJECT_PATH}"

TOKEN="${GITLAB_TOKEN:-}"
if [ -z "$TOKEN" ]; then
    TOKEN=$(glab auth token 2>/dev/null || echo "")
fi

if [ -z "$TOKEN" ]; then
    echo "Warning: No GitLab token found, skipping cleanup"
    exit 0
fi

AUTH_HEADER="PRIVATE-TOKEN: ${TOKEN}"

# Clean variables
echo "Cleaning project variables..."
curl -s -H "$AUTH_HEADER" "${API_URL}/projects/${ENCODED_PATH}/variables" | \
    jq -r '.[].key' 2>/dev/null | while read -r key; do
        echo "  Deleting variable: ${key}"
        curl -s -X DELETE -H "$AUTH_HEADER" "${API_URL}/projects/${ENCODED_PATH}/variables/${key}" > /dev/null 2>&1 || true
    done

# Clean environments
echo "Cleaning environments..."
curl -s -H "$AUTH_HEADER" "${API_URL}/projects/${ENCODED_PATH}/environments" | \
    jq -r '.[].id' 2>/dev/null | while read -r id; do
        echo "  Deleting environment: ${id}"
        curl -s -X DELETE -H "$AUTH_HEADER" "${API_URL}/projects/${ENCODED_PATH}/environments/${id}" > /dev/null 2>&1 || true
    done

# Clean test files
echo "Cleaning test files..."
curl -s -H "$AUTH_HEADER" "${API_URL}/projects/${ENCODED_PATH}/repository/tree?recursive=true" | \
    jq -r '.[] | select(.type=="blob") | .path' 2>/dev/null | \
    grep -i "formae\|test\|inttest" | while read -r filepath; do
        echo "  Deleting file: ${filepath}"
        ENCODED_FILE=$(python3 -c "import urllib.parse; print(urllib.parse.quote('${filepath}', safe=''))")
        curl -s -X DELETE -H "$AUTH_HEADER" "${API_URL}/projects/${ENCODED_PATH}/repository/files/${ENCODED_FILE}" \
            -d '{"branch":"main","commit_message":"test: cleanup"}' \
            -H "Content-Type: application/json" > /dev/null 2>&1 || true
    done

echo "clean-environment.sh: Cleanup complete"
