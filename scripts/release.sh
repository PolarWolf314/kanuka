#!/bin/bash
set -e

# 1. Get latest conventional commit version
LATEST_COMMIT_MSG=$(git log -1 --pretty=%B)

# 2. Determine version bump
if [[ "$LATEST_COMMIT_MSG" =~ BREAKING\ CHANGE|! ]]; then
  BUMP="major"
elif [[ "$LATEST_COMMIT_MSG" =~ ^feat ]]; then
  BUMP="minor"
elif [[ "$LATEST_COMMIT_MSG" =~ ^fix ]]; then
  BUMP="patch"
else
  echo "No version bump commit detected."
  exit 0
fi

# 3. Get latest tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
LATEST_VERSION=${LATEST_TAG#v}
IFS='.' read -r MAJOR MINOR PATCH <<<"$LATEST_VERSION"

case "$BUMP" in
major) VERSION="v$((MAJOR + 1)).0.0" ;;
minor) VERSION="v$MAJOR.$((MINOR + 1)).0" ;;
patch) VERSION="v$MAJOR.$MINOR.$((PATCH + 1))" ;;
esac

# 4. Create the tag locally first (don't push yet)
git tag "$VERSION"

# 5. Generate changelog
git-chglog -o CHANGELOG.md "$VERSION"

# 6. Commit the changelog
git config --local user.email "action@github.com"
git config --local user.name "GitHub Action"
git add CHANGELOG.md
git commit -m "chore: update changelog for $VERSION"

# 7. Push the commit and tag
git push
git push origin "$VERSION"

# 8. Run GoReleaser
goreleaser release --clean
