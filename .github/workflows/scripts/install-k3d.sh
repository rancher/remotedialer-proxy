#!/usr/bin/env bash
set -euo pipefail
K3D_VERSION="${K3D_VERSION:-v5.6.0}"
echo "[INFO] Installing k3d version $K3D_VERSION..."
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
k3d --version

