#!/bin/bash
# © 2025 Platform Engineering Labs Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Script to run conformance tests against a specific version of formae.
#
# Usage:
#   ./scripts/run-conformance-tests.sh [VERSION]
#
# Arguments:
#   VERSION - Optional formae version (e.g., 0.83.0). Defaults to "latest".
#
# Environment variables:
#   FORMAE_BINARY      - Path to formae binary (skips download if set)
#   FORMAE_INSTALL_PREFIX - Installation directory (default: temp directory)
#   FORMAE_TEST_FILTER - Filter tests by name pattern
#   FORMAE_TEST_TYPE   - Select test type: "all" (default), "crud", or "discovery"

set -euo pipefail

# Cross-platform sed in-place edit (macOS vs Linux)
sed_inplace() {
    if [[ "$(uname)" == "Darwin" ]]; then
        sed -i '' "$@"
    else
        sed -i "$@"
    fi
}

VERSION="${1:-latest}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# =============================================================================
# Setup Formae Binary
# =============================================================================

if [[ -n "${FORMAE_BINARY:-}" ]] && [[ -x "${FORMAE_BINARY}" ]]; then
    echo "Using FORMAE_BINARY from environment: ${FORMAE_BINARY}"
    if [[ "${VERSION}" == "latest" ]]; then
        VERSION=$("${FORMAE_BINARY}" --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        if [[ -z "${VERSION}" ]]; then
            echo "Warning: Could not extract version from FORMAE_BINARY, using 'latest'"
            VERSION="latest"
        else
            echo "Detected formae version: ${VERSION}"
        fi
    fi
else
    INSTALL_DIR=$(mktemp -d -t formae-conformance-XXXXXX)
    echo "Using temp directory: ${INSTALL_DIR}"
    trap "rm -rf ${INSTALL_DIR}" EXIT

    DETECTED_OS=$(uname | tr '[:upper:]' '[:lower:]')
    DETECTED_ARCH=$(uname -m | tr -d '_')

    if [[ "${VERSION}" == "latest" ]]; then
        echo "Resolving latest version..."
        VERSION=$(curl -s https://hub.platform.engineering/binaries/repo.json | \
            jq -r "[.Packages[] | select(.Version | index(\"-\") | not) | select(.OsArch.OS == \"${DETECTED_OS}\" and .OsArch.Arch == \"${DETECTED_ARCH}\")][0].Version")
        if [[ -z "${VERSION}" || "${VERSION}" == "null" ]]; then
            echo "Error: Could not determine latest version for ${DETECTED_OS}-${DETECTED_ARCH}"
            exit 1
        fi
    fi

    echo "Downloading formae version ${VERSION}..."
    PKGNAME="formae@${VERSION}_${DETECTED_OS}-${DETECTED_ARCH}.tgz"
    DOWNLOAD_URL="https://hub.platform.engineering/binaries/pkgs/${PKGNAME}"

    if ! curl -fsSL "${DOWNLOAD_URL}" -o "${INSTALL_DIR}/${PKGNAME}"; then
        echo "Error: Failed to download ${DOWNLOAD_URL}"
        exit 1
    fi

    echo "Extracting..."
    tar -xzf "${INSTALL_DIR}/${PKGNAME}" -C "${INSTALL_DIR}"

    FORMAE_BINARY="${INSTALL_DIR}/formae/bin/formae"
    if [[ ! -x "${FORMAE_BINARY}" ]]; then
        if [[ -x "${INSTALL_DIR}/bin/formae" ]]; then
            FORMAE_BINARY="${INSTALL_DIR}/bin/formae"
        elif [[ -x "${INSTALL_DIR}/formae" ]]; then
            FORMAE_BINARY="${INSTALL_DIR}/formae"
        else
            echo "Error: formae binary not found in ${INSTALL_DIR}"
            find "${INSTALL_DIR}" -name "formae" -type f 2>/dev/null || ls -laR "${INSTALL_DIR}"
            exit 1
        fi
    fi
fi

echo ""
echo "Using formae binary: ${FORMAE_BINARY}"
"${FORMAE_BINARY}" --version

export FORMAE_BINARY
export FORMAE_VERSION="${VERSION}"

if [[ -n "${FORMAE_TEST_FILTER:-}" ]]; then
    export FORMAE_TEST_FILTER
    echo "Test filter: ${FORMAE_TEST_FILTER}"
fi
if [[ -n "${FORMAE_TEST_TYPE:-}" ]]; then
    export FORMAE_TEST_TYPE
    echo "Test type: ${FORMAE_TEST_TYPE}"
fi

# =============================================================================
# Update and Resolve PKL Dependencies
# =============================================================================

echo ""
echo "Updating PKL dependencies for formae version ${VERSION}..."

if [[ "${VERSION}" != "latest" ]]; then
    if [[ -f "${PROJECT_ROOT}/schema/pkl/PklProject" ]]; then
        echo "Updating schema/pkl/PklProject to use formae@${VERSION}..."
        sed_inplace "s|formae/formae@[0-9a-zA-Z.\-]*\"|formae/formae@${VERSION}\"|g" "${PROJECT_ROOT}/schema/pkl/PklProject"
    fi

    if [[ -f "${PROJECT_ROOT}/testdata/PklProject" ]]; then
        echo "Updating testdata/PklProject to use formae@${VERSION}..."
        sed_inplace "s|formae/formae@[0-9a-zA-Z.\-]*\"|formae/formae@${VERSION}\"|g" "${PROJECT_ROOT}/testdata/PklProject"
    fi
fi

if [[ -f "${PROJECT_ROOT}/schema/pkl/PklProject" ]]; then
    echo "Resolving schema/pkl dependencies..."
    if ! pkl project resolve "${PROJECT_ROOT}/schema/pkl" 2>&1; then
        echo "Error: Failed to resolve schema/pkl dependencies"
        exit 1
    fi
fi

if [[ -f "${PROJECT_ROOT}/testdata/PklProject" ]]; then
    echo "Resolving testdata dependencies..."
    if ! pkl project resolve "${PROJECT_ROOT}/testdata" 2>&1; then
        echo "Error: Failed to resolve testdata dependencies"
        exit 1
    fi
fi

echo "PKL dependencies resolved successfully"

# =============================================================================
# Run Conformance Tests
# =============================================================================
echo ""
echo "Running conformance tests..."
cd "${PROJECT_ROOT}"
go test -tags=conformance -v -timeout 30m ./...
