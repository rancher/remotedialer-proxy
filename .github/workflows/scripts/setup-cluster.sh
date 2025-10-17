#!/usr/bin/env bash
set -euo pipefail
CLUSTER_NAME="${CLUSTER_NAME:-rdproxy-ci}"
K3S_VERSION="${K3S_VERSION:-v1.27.4-k3s1}"
ARCH="${ARCH:-amd64}"

if ! k3d cluster list | grep -q "$CLUSTER_NAME"; then
  echo "[INFO] Creating k3d cluster $CLUSTER_NAME..."
  k3d cluster create $CLUSTER_NAME --image rancher/k3s:$K3S_VERSION --agents 1 --wait
else
  echo "[INFO] k3d cluster $CLUSTER_NAME already exists."
fi

