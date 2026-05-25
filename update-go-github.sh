#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 <vNN>  (e.g. $0 v80)" >&2
  exit 1
}

die() {
  echo "error: $*" >&2
  exit 1
}

confirm() {
  read -r -p "Ready to $1 (yubikey). Press Enter to continue..."
}

# --- Validate arguments ---
[[ $# -eq 1 ]] || usage
VERSION="$1"
[[ "$VERSION" =~ ^v[0-9]+$ ]] || die "argument must match v<number> (e.g. v80), got: $VERSION"
NUM="${VERSION#v}"

# --- Validate working tree ---
[[ -z "$(git status --porcelain --untracked-files=no)" ]] || die "working tree is not clean; commit or stash changes first"
[[ "$(git branch --show-current)" == "main" ]] || die "must be on main branch"

# --- Derive module base path (strip any existing /vNN suffix) ---
MODULE_BASE="$(head -1 go.mod | awk '{print $2}' | sed -E 's|/v[0-9]+$||')"
echo "Module base: ${MODULE_BASE}"

# ============================================================
# Phase 1: Update main
# ============================================================
echo "==> Phase 1: Updating main branch"

echo "  -> Updating go-github to ${VERSION}..."
go get "github.com/google/go-github/${VERSION}@latest"

echo "  -> Updating go-github imports in .go files..."
find . -name '*.go' -not -path './vendor/*' -exec \
  sed -i '' -E "s|github\\.com/google/go-github/v[0-9]+/github|github.com/google/go-github/${VERSION}/github|g" {} +

echo "  -> Updating all dependencies..."
go get -u ./...
go mod tidy

echo "  -> Verifying main..."
go build ./...
go vet ./...
go test ./...

if [[ -n "$(git status --porcelain --untracked-files=no)" ]]; then
  git add -u
  confirm "commit main updates"
  git commit -m "chore(deps): bump go-github to ${VERSION} and update all dependencies"
else
  echo "  -> No changes on main; skipping commit."
fi

# ============================================================
# Phase 2: Create ${VERSION} branch
# ============================================================
echo "==> Phase 2: Creating ${VERSION} branch"

if git show-ref --verify --quiet "refs/heads/${VERSION}"; then
  die "branch ${VERSION} already exists locally; this script only handles new major version creation"
fi

git checkout -b "${VERSION}"

echo "  -> Updating module path..."
sed -i '' -E "1s|^module .*|module ${MODULE_BASE}/${VERSION}|" go.mod

echo "  -> Updating internal imports..."
find . -name '*.go' -exec \
  sed -i '' -E "s|${MODULE_BASE}(/v[0-9]+)?|${MODULE_BASE}/${VERSION}|g" {} +

echo "  -> Running go mod tidy..."
go mod tidy

echo "  -> Verifying ${VERSION} branch..."
go build ./...
go vet ./...
go test ./...

git add -u
confirm "commit ${VERSION} module"
git commit -m "chore: create ${VERSION} module tracking go-github/${VERSION}"

# --- Tag ---
LATEST_TAG="$(git tag -l "${VERSION}.*" | sort -V | tail -1 || true)"
if [[ -z "$LATEST_TAG" ]]; then
  NEW_TAG="${VERSION}.0.0"
else
  # Bump patch version
  PATCH="${LATEST_TAG##*.}"
  PREFIX="${LATEST_TAG%.*}"
  NEW_TAG="${PREFIX}.$((PATCH + 1))"
fi

confirm "create signed tag ${NEW_TAG}"
git tag -s -a "${NEW_TAG}" -m "Release ${NEW_TAG} tracking go-github/${VERSION}"

# ============================================================
# Phase 3: Return to main
# ============================================================
echo "==> Phase 3: Returning to main"
git checkout main

echo ""
echo "Done! Created branch ${VERSION} with tag ${NEW_TAG}"
echo "To push: git push origin main ${VERSION} ${NEW_TAG}"
