#!/bin/sh

set -e

RANCHER_REPO_DIR=$1
# File to write the changes to
CHANGES_FILE=${2:-/dev/null}

DEPS_TO_SYNC="
  github.com/rancher/remotedialer
  github.com/rancher/dynamiclistener
  github.com/rancher/wrangler/v3
"

if [ -z "$RANCHER_REPO_DIR" ]; then
  usage
  exit 1
fi

usage() {
  echo "$0 <path to rancher repository> [<path to write dependency changes to>]"
}

update_dep() {
  module=$1
  old_version=$2
  new_version=$3

  echo "Version mismatch for $module (rancher=$new_version, rdp=$old_version) detected"
  go mod edit -require="$module@$new_version"
  printf '**%s**\n`%s` => `%s`\n' "$module" "$old_version" "$new_version" >> "$CHANGES_FILE"
}

rancher_deps=$(cd "$RANCHER_REPO_DIR" && go mod graph)
remotedialerproxy_deps=$(go mod graph)

for dep in $DEPS_TO_SYNC; do
  if ! rancher_version=$(echo "$rancher_deps" | grep "^$dep@\w*\S"); then
    continue
  fi

  if ! rdp_version=$(echo "$remotedialerproxy_deps" | grep "^$dep@\w*\S"); then
    continue
  fi

  rancher_version=$(echo "$rancher_version" | head -n 1 | cut -d' ' -f1 | cut -d@ -f2)
  rdp_version=$(echo "$rdp_version" | head -n 1 | cut -d' ' -f1 | cut -d@ -f2)
  if [ "$rancher_version" = "$rdp_version" ]; then
    continue
  fi

  # If the rancher version is not newer, we should not update.
  latest_version=$(printf '%s\n%s' "$rancher_version" "$rdp_version" | sort --version-sort | tail -n1)
  if [ "$latest_version" != "$rancher_version" ]; then
    echo "Skipping update for $dep because rancher version ($rancher_version) is not newer than rdp version ($rdp_version)"
    continue
  fi

  update_dep "$dep" "$rdp_version" "$rancher_version"
done

echo "Running go mod tidy"
go mod tidy
