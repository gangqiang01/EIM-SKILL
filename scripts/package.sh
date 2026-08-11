#!/usr/bin/env bash
# Bundle the compiled eim-skill binaries plus docs/skills/config template
# into a single release tarball: DeviceOn-EIM-SKILL_<version>.tar.gz
#
# Usage: scripts/package.sh [version]
#   version defaults to the repo-root VERSION file's contents.
#
# Expects dist/eim-skill, dist/eim-skill-arm64, and dist/eim-skill.exe to
# already exist (run "make cross" first, or just "make package" which does
# both steps).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

VERSION="${1:-$(cat VERSION 2>/dev/null || echo dev)}"
DIST_DIR="dist"
PKG_NAME="DeviceOn-EIM-SKILL_${VERSION}"
STAGE_DIR="${DIST_DIR}/${PKG_NAME}"

binaries=()
for b in "${DIST_DIR}/eim-skill" "${DIST_DIR}/eim-skill-arm64" "${DIST_DIR}/eim-skill.exe"; do
	[ -f "$b" ] && binaries+=("$b")
done
if [ ${#binaries[@]} -eq 0 ]; then
	echo "error: no compiled binaries found in ${DIST_DIR}/ - run 'make cross' first" >&2
	exit 1
fi

echo "Packaging ${PKG_NAME}.tar.gz ..."
rm -rf "${STAGE_DIR}"
mkdir -p "${STAGE_DIR}"

cp README.md README-zh.md "${STAGE_DIR}/"
cp -r skills "${STAGE_DIR}/"
cp "${binaries[@]}" "${STAGE_DIR}/"

# Also copy the binaries into skills/scripts/, a sibling of every skill
# folder - this is what each SKILL.md's "../scripts/eim-skill..." example
# commands actually resolve to when a skill runner (e.g. workbuddy)
# deploys the skills/ directory on its own, without the rest of this
# tarball (so skills/ must be fully self-contained, not rely on the
# top-level binary copies above).
mkdir -p "${STAGE_DIR}/skills/scripts"
cp "${binaries[@]}" "${STAGE_DIR}/skills/scripts/"

# Ship a placeholder config template, never the real conf/config.yaml
# (which may hold a live internal server address/credentials) - the
# recipient copies this to config.yaml and fills in their own values.
mkdir -p "${STAGE_DIR}/conf"
cp conf/config.yaml "${STAGE_DIR}/conf/"

tar -czf "${PKG_NAME}.tar.gz" -C "${DIST_DIR}" "${PKG_NAME}"
rm -rf "${STAGE_DIR}"

echo "Created ${PKG_NAME}.tar.gz"
tar -tzf "${PKG_NAME}.tar.gz"
