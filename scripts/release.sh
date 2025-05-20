#!/bin/bash
set -e

# 1. Get the squash commit message - focusing on the PR title/description
SQUASH_COMMIT_MSG=$(git log -1 --pretty=%B | head -1)
echo "Analyzing commit message: $SQUASH_COMMIT_MSG"

# Also get the PR description (if available) - typically first paragraph of squash commit
PR_DESCRIPTION=$(git log -1 --pretty=%B | awk 'NR>1 && NR<=3 {print}')

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

# 4. Generate changelog entry from the PR title/description
echo "Generating changelog entry from the PR title/description..."
CHANGELOG_ENTRIES=""

# First, use the PR title (first line of squash commit)
if [[ "$SQUASH_COMMIT_MSG" =~ ^(feat|fix|docs|style|refactor|perf|test|build|ci|chore).*:\ +(.*) ]]; then
  # Extract the type (feat, fix, etc.)
  TYPE=$(echo "$SQUASH_COMMIT_MSG" | sed -E 's/^([^(:]+)(\([^)]*\))?:.*/\1/')
  # Extract the scope if present (the part in parentheses)
  SCOPE=$(echo "$SQUASH_COMMIT_MSG" | sed -E 's/^[^(:]+(\([^)]*\))?.*/\1/; s/^[^(]//; s/^$//')
  if [[ -n "$SCOPE" ]]; then
    SCOPE="($SCOPE)"
  fi
  # Extract the description (everything after the colon and space)
  DESC=$(echo "$SQUASH_COMMIT_MSG" | sed -E 's/^[^:]+:\ +//')

  # Format nicely for changelog
  if [[ -n "$SCOPE" ]]; then
    CHANGELOG_ENTRIES+="* ${TYPE}${SCOPE}: ${DESC}\n"
  else
    CHANGELOG_ENTRIES+="* ${TYPE}: ${DESC}\n"
  fi
else
  # If no conventional commit format, use the title as is
  CHANGELOG_ENTRIES+="* ${SQUASH_COMMIT_MSG}\n"
fi

# Add PR description if available and meaningful
if [[ -n "$PR_DESCRIPTION" && ${#PR_DESCRIPTION} -gt 10 ]]; then
  CHANGELOG_ENTRIES+="\n  ${PR_DESCRIPTION}\n"
fi

# 5. Create a temporary changelog entry for this version
TEMP_CHANGELOG=$(mktemp)
cat >"$TEMP_CHANGELOG" <<EOF
## ${VERSION} ($(date +%Y-%m-%d))

$(echo -e "$CHANGELOG_ENTRIES")
EOF

cat "$TEMP_CHANGELOG"

# 6. Create the tag locally first
git tag "$VERSION"

# If git-chglog is available, use it and merge our entries, otherwise use our simple format
if command -v git-chglog >/dev/null; then
  echo "Using git-chglog to generate changelog..."
  git-chglog -o CHANGELOG.md "$VERSION"

  # Optional: You could merge your parsed entries into the git-chglog output
  # for better coverage of the squashed commits
else
  echo "git-chglog not found, using simple changelog format..."
  # If CHANGELOG.md exists, insert new entries at the top, otherwise create it
  if [ -f CHANGELOG.md ]; then
    cat "$TEMP_CHANGELOG" <(echo) CHANGELOG.md >CHANGELOG.md.new
    mv CHANGELOG.md.new CHANGELOG.md
  else
    cat >CHANGELOG.md <<EOF
# Changelog

$(cat "$TEMP_CHANGELOG")
EOF
  fi
fi

rm "$TEMP_CHANGELOG"

# 7. Commit the changelog
git config --local user.email "action@github.com"
git config --local user.name "GitHub Action"
git add CHANGELOG.md
git commit -m "chore: update changelog for $VERSION"

# 8. Retag the newest commit, and push
git tag -f "$VERSION"
git push
git push origin "$VERSION"

# 9. Run GoReleaser
goreleaser release --clean
