#!/bin/bash
# Prepares charts/remotedialer-proxy for the ship-staged action of rancher/charts:
# copies it to <output dir>/remotedialer-proxy with the appVersion and the image
# tag set to the release tag, and prints the final chart version on stdout. The
# action stamps, checks, packages and pushes that folder.
#
# The chart version is <chart major>.0.<remotedialer-proxy patch>+up<remotedialer-proxy version>,
# with the chart major read from the "Chart Major Version" column of VERSION.md
# for the remotedialer-proxy minor of the tag.
#
# Usage: prepare-chart.sh <tag> <output dir>
#   e.g. prepare-chart.sh v0.10.0-rc.1 dist/oci
#
# Run from the repository root.
set -euo pipefail

TAG=${1:?usage: $0 <tag> <output dir>}
OUT_DIR=${2:?usage: $0 <tag> <output dir>}

if ! [[ "$TAG" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)(-[0-9A-Za-z.]+)?$ ]]; then
  echo "Error: tag $TAG must look like vX.Y.Z or vX.Y.Z-rc.N" >&2
  exit 1
fi
MINOR="v${BASH_REMATCH[1]}.${BASH_REMATCH[2]}"
PATCH="${BASH_REMATCH[3]}"
APP_VERSION="${TAG#v}"

# | remotedialer-proxy Branch | remotedialer-proxy Minor version | Rancher Minor Version | Chart Major Version |
CHART_MAJOR=$(awk -F'|' -v minor="$MINOR" '{gsub(/ /, "", $3); gsub(/ /, "", $5)} $3 == minor {print $5}' VERSION.md)
if ! [[ "$CHART_MAJOR" =~ ^[0-9]+$ ]]; then
  echo "Error: no chart major for $MINOR in VERSION.md (got '$CHART_MAJOR')" >&2
  exit 1
fi
CHART_VERSION="${CHART_MAJOR}.0.${PATCH}+up${APP_VERSION}"

CHART_DIR="$OUT_DIR/remotedialer-proxy"
rm -rf "$CHART_DIR"
mkdir -p "$OUT_DIR"
cp -r charts/remotedialer-proxy "$CHART_DIR"

# stamp keeps the helm.sh/images annotation that Chart.yaml already has and refuses
# an image tagged "latest", and it changes neither the appVersion nor the tag of
# values.yaml. The annotation is "latest" in the source and the values tag is empty
# (the deployment then falls back to the appVersion, which has no "v" unlike the
# image tags), so set all of them here.
sed -i \
  -e "s|^appVersion:.*|appVersion: ${APP_VERSION}|" \
  -e "s|image: rancher/remotedialer-proxy:latest|image: rancher/remotedialer-proxy:${TAG}|" \
  "$CHART_DIR/Chart.yaml"
sed -i -e "s|^  tag: .*|  tag: ${TAG}|" "$CHART_DIR/values.yaml"

echo "$CHART_VERSION"
