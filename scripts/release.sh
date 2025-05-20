#!/bin/bash
set -e

# 1. Get the squash commit message - focusing on the PR title/description
SQUASH_COMMIT_MSG=$(git log -1 --pretty=%B | head -1)
echo "Analyzing commit message: $SQUASH_COMMIT_MSG"

# 2. Determine version bump based on the PR title
if [[ "$SQUASH_COMMIT_MSG" =~ BREAKING\ CHANGE|! ]]; then
  BUMP="major"
  echo "BREAKING CHANGE detected, will bump major version"
elif [[ "$SQUASH_COMMIT_MSG" =~ feat:|^feat ]]; then
  BUMP="minor"
  echo "Feature detected, will bump minor version"
elif [[ "$SQUASH_COMMIT_MSG" =~ fix:|^fix ]]; then
  BUMP="patch"
  echo "Fix detected, will bump patch version"
else
  echo "No version bump commit detected."
  exit 0
fi

# 3. Get latest tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
LATEST_VERSION=${LATEST_TAG#v}
echo "Latest version: $LATEST_VERSION"

IFS='.' read -r MAJOR MINOR PATCH <<<"$LATEST_VERSION"
case "$BUMP" in
major) VERSION="v$((MAJOR + 1)).0.0" ;;
minor) VERSION="v$MAJOR.$((MINOR + 1)).0" ;;
patch) VERSION="v$MAJOR.$MINOR.$((PATCH + 1))" ;;
esac

echo "New version will be: $VERSION"

# 4. Create the tag locally first
git tag "$VERSION"

# 5. Create changelog
git-chglog -o CHANGELOG.md "$VERSION"

# 6. Commit the changelog
git config --local user.email "action@github.com"
git config --local user.name "GitHub Action"
git add CHANGELOG.md
git commit -m "chore: update changelog for $VERSION"

# 7. Retag the newest commit, and push
git tag -f "$VERSION"
git push
git push origin "$VERSION"

# 8. Run GoReleaser
goreleaser release --clean
