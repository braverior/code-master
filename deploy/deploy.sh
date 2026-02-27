#!/bin/bash
# ============================================================
# CodeMaster K8s One-Click Deploy
# Usage:
#   ./deploy.sh build     - Build Docker image
#   ./deploy.sh push      - Push to registry
#   ./deploy.sh apply     - Apply K8s manifests
#   ./deploy.sh all       - Build + Push + Apply
#   ./deploy.sh delete    - Remove from K8s
# ============================================================
set -e

# ---------- Configuration ----------
IMAGE_NAME="${IMAGE_NAME:-codemaster}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
REGISTRY="${REGISTRY:-}"                  # e.g. "registry.example.com/team"
FULL_IMAGE="${REGISTRY:+${REGISTRY}/}${IMAGE_NAME}:${IMAGE_TAG}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
K8S_DIR="$SCRIPT_DIR/k8s"

# ---------- Functions ----------
build() {
  echo ">>> Building Docker image: ${FULL_IMAGE}"
  docker build -t "${FULL_IMAGE}" "$PROJECT_ROOT"
  echo ">>> Build complete"
}

push() {
  if [ -z "$REGISTRY" ]; then
    echo ">>> REGISTRY not set, skipping push (local image only)"
    return
  fi
  echo ">>> Pushing ${FULL_IMAGE}"
  docker push "${FULL_IMAGE}"
  echo ">>> Push complete"
}

apply() {
  echo ">>> Applying K8s manifests"

  # Update image in deployment if using a registry
  if [ -n "$REGISTRY" ]; then
    echo "    Image: ${FULL_IMAGE}"
    sed -i.bak "s|image: codemaster:latest|image: ${FULL_IMAGE}|" "$K8S_DIR/deployment.yaml"
  fi

  kubectl apply -f "$K8S_DIR/namespace.yaml"
  kubectl apply -f "$K8S_DIR/secret.yaml"
  kubectl apply -f "$K8S_DIR/configmap.yaml"
  kubectl apply -f "$K8S_DIR/pvc.yaml"
  kubectl apply -f "$K8S_DIR/mysql.yaml"
  kubectl apply -f "$K8S_DIR/redis.yaml"
  kubectl apply -f "$K8S_DIR/deployment.yaml"
  kubectl apply -f "$K8S_DIR/service.yaml"
  kubectl apply -f "$K8S_DIR/ingress.yaml"

  # Restore original deployment.yaml
  if [ -f "$K8S_DIR/deployment.yaml.bak" ]; then
    mv "$K8S_DIR/deployment.yaml.bak" "$K8S_DIR/deployment.yaml"
  fi

  echo ">>> Waiting for rollout..."
  kubectl -n codemaster rollout status deployment/mysql --timeout=120s
  kubectl -n codemaster rollout status deployment/redis --timeout=60s
  kubectl -n codemaster rollout status deployment/codemaster --timeout=180s

  echo ""
  echo ">>> Deploy complete! Pod status:"
  kubectl -n codemaster get pods
  echo ""
  echo ">>> Services:"
  kubectl -n codemaster get svc
}

delete() {
  echo ">>> Deleting CodeMaster from K8s..."
  kubectl delete namespace codemaster --ignore-not-found
  echo ">>> Deleted"
}

# ---------- Main ----------
case "${1:-}" in
  build)  build ;;
  push)   push ;;
  apply)  apply ;;
  all)    build; push; apply ;;
  delete) delete ;;
  *)
    echo "Usage: $0 {build|push|apply|all|delete}"
    echo ""
    echo "Environment variables:"
    echo "  REGISTRY    - Docker registry (e.g. registry.example.com/team)"
    echo "  IMAGE_NAME  - Image name (default: codemaster)"
    echo "  IMAGE_TAG   - Image tag (default: latest)"
    exit 1
    ;;
esac
