#!/bin/bash
set -e

# 1. Get the squash commit message - this will be our source of conventional commit info
SQUASH_COMMIT_MSG=$(git log -1 --pretty=%B)
echo "Analyzing commit message: $SQUASH_COMMIT_MSG"

# Extract conventional commit information from the squash message
# Look for conventional commit patterns in the squash message
# Typically a squash message contains multiple commit messages separated by lines

# 2. Determine version bump based on the entire squash message
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

# 4. Parse the squash commit message to generate changelog entries
echo "Generating changelog entries from squash commit..."
CHANGELOG_ENTRIES=""

# Each line of the squash message might represent a separate conventional commit
# Format: Extract commit type and description for changelog
while IFS= read -r line; do
  # Skip empty lines and merge commit lines
  if [[ -z "$line" || "$line" =~ ^Merge\ |pull\ request ]]; then
    continue
  fi

  # Try to extract type (feat, fix, etc) and description
  if [[ "$line" =~ ^(feat|fix|docs|style|refactor|perf|test|build|ci|chore).*:\ +(.*) ]]; then
    # Extract the type (feat, fix, etc.)
    TYPE=$(echo "$line" | sed -E 's/^([^(:]+)(\([^)]*\))?:.*/\1/')
    # Extract the scope if present (the part in parentheses)
    SCOPE=$(echo "$line" | sed -E 's/^[^(:]+(\([^)]*\))?.*/\1/; s/^[^(]//; s/^$//')
    if [[ -n "$SCOPE" ]]; then
      SCOPE="($SCOPE)"
    fi
    # Extract the description (everything after the colon and space)
    DESC=$(echo "$line" | sed -E 's/^[^:]+:\ +//')

    # Format nicely for changelog
    if [[ -n "$SCOPE" ]]; then
      CHANGELOG_ENTRIES+="* ${TYPE}${SCOPE}: ${DESC}\n"
    else
      CHANGELOG_ENTRIES+="* ${TYPE}: ${DESC}\n"
    fi
  else
    # For lines without the conventional format, include them as-is if they seem meaningful
    if [[ ${#line} -gt 5 && ! "$line" =~ ^[#\*] ]]; then
      CHANGELOG_ENTRIES+="* ${line}\n"
    fi
  fi
done <<<"$SQUASH_COMMIT_MSG"

# If we couldn't extract meaningful entries, use the first line as a fallback
if [[ -z "$CHANGELOG_ENTRIES" ]]; then
  FIRST_LINE=$(echo "$SQUASH_COMMIT_MSG" | head -n 1)
  CHANGELOG_ENTRIES="* ${FIRST_LINE}\n"
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
