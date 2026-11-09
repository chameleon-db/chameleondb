#!/usr/bin/env bash
#
# Prepare and publish a release: bump the version in chameleon-core/Cargo.toml,
# commit it, create the two release tags and push them.
#
# Usage: scripts/tag-release.sh [VERSION]      (or: make tag [VERSION=X.Y.Z])
#
# Without VERSION the script proposes the next patch version; press Enter to
# accept it. The release notes must already be on main: the script refuses to
# tag a commit whose CHANGELOG.md and RELEASE.md do not mention the version.

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/.." && pwd)"
CORE_DIR="$REPO_ROOT/chameleon-core"
CARGO_TOML="$CORE_DIR/Cargo.toml"
RELEASE_BRANCH="${RELEASE_BRANCH:-main}"
REMOTE="${REMOTE:-origin}"

die() {
  echo "error: $*" >&2
  exit 1
}

cd "$REPO_ROOT"

# --- Repository state ---------------------------------------------------------

[[ -z "$(git status --porcelain)" ]] ||
  die "the working tree has uncommitted changes. Commit or stash them before creating a release."

branch="$(git symbolic-ref --quiet --short HEAD)" ||
  die "HEAD is detached. Check out $RELEASE_BRANCH first."
[[ "$branch" == "$RELEASE_BRANCH" ]] ||
  die "releases are made from '$RELEASE_BRANCH', but the current branch is '$branch'."

git fetch --quiet --tags "$REMOTE"
local_head="$(git rev-parse HEAD)"
remote_head="$(git rev-parse "$REMOTE/$RELEASE_BRANCH")"
if [[ "$local_head" != "$remote_head" ]]; then
  if git merge-base --is-ancestor "$local_head" "$remote_head"; then
    die "$branch is behind $REMOTE/$RELEASE_BRANCH. Run 'git pull' first."
  elif git merge-base --is-ancestor "$remote_head" "$local_head"; then
    die "$branch has commits that are not on $REMOTE/$RELEASE_BRANCH. Merge them through a pull request first."
  else
    die "$branch and $REMOTE/$RELEASE_BRANCH have diverged."
  fi
fi

# --- Version ------------------------------------------------------------------

current_version="$(awk '
  $0 == "[package]" { in_package = 1; next }
  in_package && /^version = "/ {
    sub(/^version = "/, "")
    sub(/"$/, "")
    print
    exit
  }
  in_package && /^\[/ { exit }
' "$CARGO_TOML")"

[[ -n "$current_version" ]] || die "could not find the package version in $CARGO_TOML."

if [[ "$current_version" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)(.*)$ ]]; then
  default_version="${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.$((BASH_REMATCH[3] + 1))${BASH_REMATCH[4]}"
else
  die "the current version is not SemVer: $current_version"
fi

requested_version="${1:-}"
if [[ -z "$requested_version" ]]; then
  echo "Current version: $current_version"
  read -r -p "New version [v$default_version]: " requested_version
  requested_version="${requested_version:-v$default_version}"
fi
requested_version="${requested_version#v}"

[[ "$requested_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.+][0-9A-Za-z.-]+)?$ ]] ||
  die "invalid version: $requested_version"
[[ "$requested_version" != "$current_version" ]] ||
  die "the new version is the same as the current version."

release_tag="v$requested_version"
component_tag="chameleon/$release_tag"

for tag in "$release_tag" "$component_tag"; do
  if git rev-parse --quiet --verify "refs/tags/$tag" >/dev/null; then
    die "the tag already exists: $tag"
  fi
done

# --- Release notes must be on main before tagging ------------------------------

core_version="${requested_version%%[-+]*}"
grep -Eq "^## \[(v$requested_version|v$core_version)\]" CHANGELOG.md ||
  die "CHANGELOG.md has no '## [v$core_version]' section. Add the release notes and merge them first."
grep -Fq "[$release_tag]" RELEASE.md ||
  die "RELEASE.md does not mention [$release_tag]. Update it and merge it first."

# --- Bump, commit, tag ---------------------------------------------------------

backup_dir="$(mktemp -d)"
cp "$CARGO_TOML" "$backup_dir/Cargo.toml"
restore_on_failure=1
cleanup() {
  status=$?
  if [[ "$status" -ne 0 && "$restore_on_failure" -eq 1 ]]; then
    cp "$backup_dir/Cargo.toml" "$CARGO_TOML"
    echo "Cargo.toml was restored because the release could not be prepared." >&2
  fi
  rm -rf "$backup_dir"
  exit "$status"
}
trap cleanup EXIT

awk -v new_version="$requested_version" '
  $0 == "[package]" { in_package = 1 }
  in_package && !updated && /^version = "/ {
    print "version = \"" new_version "\""
    updated = 1
    next
  }
  in_package && /^\[/ && $0 != "[package]" { in_package = 0 }
  { print }
' "$CARGO_TOML" > "$backup_dir/Cargo.toml.new"
mv "$backup_dir/Cargo.toml.new" "$CARGO_TOML"

echo "Checking the Cargo manifest..."
(cd "$CORE_DIR" && cargo check --quiet)

echo
echo "Prepared change:"
git diff -- "$CARGO_TOML"
echo

read -r -p "Create the commit and both tags now? [Y/n]: " create_release
create_release="${create_release:-y}"

if [[ ! "$create_release" =~ ^[Yy]$ ]]; then
  restore_on_failure=0
  git checkout -- "$CARGO_TOML"
  echo "Nothing was committed or tagged. Cargo.toml was restored."
  exit 0
fi

git add "$CARGO_TOML"
if git ls-files --error-unmatch "$CORE_DIR/Cargo.lock" >/dev/null 2>&1; then
  git add "$CORE_DIR/Cargo.lock"
fi
git commit --quiet -m "chore: release $release_tag"
restore_on_failure=0
git tag "$release_tag"
git tag "$component_tag"

echo
echo "Commit and tags created:"
echo "  $release_tag"
echo "  $component_tag"
echo

# --- Push ---------------------------------------------------------------------

read -r -p "Push the branch and the tags? [y/N]: " push_release
if [[ ! "$push_release" =~ ^[Yy]$ ]]; then
  echo "To publish the release:"
  echo "  git push $REMOTE HEAD:$branch"
  echo "  git push $REMOTE $release_tag $component_tag"
  exit 0
fi

git push "$REMOTE" "HEAD:$branch"
git push "$REMOTE" "$release_tag" "$component_tag"

remote_commit="$(git ls-remote "$REMOTE" "refs/heads/$branch" | awk '{print $1}')"
remote_release_tag="$(git ls-remote "$REMOTE" "refs/tags/$release_tag" | awk '{print $1}')"
remote_component_tag="$(git ls-remote "$REMOTE" "refs/tags/$component_tag" | awk '{print $1}')"
if [[ "$remote_commit" != "$(git rev-parse HEAD)" ||
  "$remote_release_tag" != "$(git rev-parse "$release_tag")" ||
  "$remote_component_tag" != "$(git rev-parse "$component_tag")" ]]; then
  die "the push finished, but the remote references could not all be verified."
fi
echo "Branch and tags verified on $REMOTE."
echo "The release workflow starts from the $release_tag tag."
