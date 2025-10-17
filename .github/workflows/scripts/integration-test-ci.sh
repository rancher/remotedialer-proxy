#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-rdproxy-ci}"
NAMESPACE="${NAMESPACE:-rdproxy-test}"
IMAGE_NAME="${IMAGE_NAME:-remotedialer-proxy:test}"
CLIENT_IMAGE_NAME="${CLIENT_IMAGE_NAME:-remotedialer-proxy-client:test}"
CHART_DIR="${CHART_DIR:-charts/remotedialer-proxy}"

# --- Create unique tags for this test run ---
UNIQUE_TAG="$(date +%s)"
UNIQUE_IMAGE_NAME="${IMAGE_NAME}-${UNIQUE_TAG}"
UNIQUE_CLIENT_IMAGE_NAME="${CLIENT_IMAGE_NAME}-${UNIQUE_TAG}"

docker tag "${IMAGE_NAME}" "${UNIQUE_IMAGE_NAME}"
docker tag "${CLIENT_IMAGE_NAME}" "${UNIQUE_CLIENT_IMAGE_NAME}"

# Update IMAGE_NAME vars to use the unique tags
IMAGE_NAME="${UNIQUE_IMAGE_NAME}"
CLIENT_IMAGE_NAME="${UNIQUE_CLIENT_IMAGE_NAME}"

echo "Using unique image tags for this run:"
echo "  ${IMAGE_NAME}"
echo "  ${CLIENT_IMAGE_NAME}"

# --- DEBUG INFO ---
echo "Runner architecture: $(uname -m)"
echo "Inspecting built Docker image: ${IMAGE_NAME}"
docker image inspect "${IMAGE_NAME}" | grep '"Architecture"' || echo "Image inspection failed for ${IMAGE_NAME}"
echo "Inspecting built Docker image: ${CLIENT_IMAGE_NAME}"
docker image inspect "${CLIENT_IMAGE_NAME}" | grep '"Architecture"' || echo "Image inspection failed for ${CLIENT_IMAGE_NAME}"
echo "--- END DEBUG INFO ---"

# Import image into k3d
k3d image import $IMAGE_NAME -c $CLUSTER_NAME
k3d image import $CLIENT_IMAGE_NAME -c $CLUSTER_NAME

# Create namespace
kubectl create ns $NAMESPACE || true

# Create TLS secret
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -sha256 -days 365 -nodes -subj "/CN=rdproxy.rdproxy-test.svc"
kubectl -n $NAMESPACE create secret tls dummy --cert=cert.pem --key=key.pem

# Apply RBAC rules
kubectl apply -f .github/workflows/manifests/rbac.yaml

# Deploy remotedialer-proxy directly (no Helm, no Rancher dependency)
sed "s|IMAGE_PLACEHOLDER|${IMAGE_NAME}|g" .github/workflows/manifests/rdproxy-deployment.yaml | kubectl apply -f -
kubectl apply -f .github/workflows/manifests/rdproxy-service.yaml

# Wait for deployment
if ! kubectl -n $NAMESPACE rollout status deploy/rdproxy --timeout=120s; then
  echo "Deployment failed. Dumping debug info."
  kubectl -n $NAMESPACE get deployment rdproxy
  kubectl -n $NAMESPACE describe pod -l app=rdproxy
  echo "--- CURRENT POD LOGS ---"
  kubectl -n $NAMESPACE logs -l app=rdproxy || true
  echo "--- PREVIOUS POD LOGS ---"
  kubectl -n $NAMESPACE logs --previous -l app=rdproxy || true
  exit 1
fi

echo "Deployment successful. Running HTTP check."
# Run test pod (HTTP check)
if ! kubectl -n $NAMESPACE run test-client --rm -i --restart=Never --image=curlimages/curl -- curl -sL --insecure -o /dev/null https://rdproxy:8443/connect; then
  echo "HTTP check failed. Dumping debug info."
  kubectl -n $NAMESPACE describe pod -l app=rdproxy
  echo "--- RDPROXY POD LOGS ---"
  kubectl -n $NAMESPACE logs -l app=rdproxy || true
  exit 1
fi
echo "HTTP check passed."

# Deploy a reliable TCP echo server (busybox + inetd)
kubectl -n $NAMESPACE apply -f - <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: echo-server
  labels:
    app: echo-server
spec:
  containers:
  - name: echo
    image: busybox
    command:
      - /bin/sh
      - -c
      - "echo '12345 stream tcp nowait root /bin/cat cat' > /etc/inetd.conf && exec inetd -f"
  restartPolicy: Never
EOF

# Wait for echo server pod to be ready
kubectl -n $NAMESPACE wait --for=condition=Ready pod/echo-server --timeout=60s

# Expose the echo server
kubectl -n $NAMESPACE expose pod echo-server --port=12345 --target-port=12345 --name=echo-server

# Deploy the proxy client
echo "Deploying proxy client..."
kubectl -n $NAMESPACE run proxy-client \
  --image=$CLIENT_IMAGE_NAME \
  --restart=Never \
  --env="NAMESPACE=$NAMESPACE" \
  --env="LABEL=app=rdproxy" \
  --env="CERT_SECRET_NAME=dummy" \
  --env="CERT_SERVER_NAME=rdproxy.rdproxy-test.svc" \
  --env="CONNECT_SECRET=dummy" \
  --env="CONNECT_URL=wss://127.0.0.1:8443/connect" \
  --env="PORTS=8443:8443" \
  --env="FAKE_IMPERATIVE_API_ADDR=0.0.0.0:12345"

# Wait for the client to start
echo "Waiting for proxy-client to start..."
if ! kubectl -n $NAMESPACE wait --for=condition=Ready pod/proxy-client --timeout=60s; then
    echo "Proxy client failed to become ready. Dumping logs."
    kubectl -n $NAMESPACE logs proxy-client || true
    exit 1
fi

echo "Proxy client is ready. Waiting for it to connect to the server..."
sleep 10

echo "--- PROXY CLIENT LOGS (after connect wait) ---"
kubectl -n $NAMESPACE logs proxy-client || true

# Test TCP proxying through remotedialer-proxy
echo "Running TCP proxy test..."
if ! kubectl -n $NAMESPACE run proxy-tester --rm -i --restart=Never --image=busybox -- /bin/sh -c 'echo hello-proxy | nc -w 5 rdproxy 6666 | grep hello-proxy'; then
  echo "TCP proxy test failed. Dumping debug info."
  echo "--- RDPROXY POD LOGS (after test fail) ---"
  kubectl -n $NAMESPACE logs -l app=rdproxy || true
  echo "--- PROXY CLIENT POD LOGS (after test fail) ---"
  kubectl -n $NAMESPACE logs proxy-client || true
  exit 1
fi
echo "TCP proxy test passed. Integration test successful!"
