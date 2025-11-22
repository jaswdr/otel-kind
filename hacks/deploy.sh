#!/bin/bash
set -e

NAMESPACE="${NAMESPACE:-lgtm}"
CLUSTER_NAME="${CLUSTER_NAME:-otel-cluster}"

echo "========================================="
echo "LGTM Stack Deployment"
echo "========================================="
echo "Namespace: $NAMESPACE"
echo "Cluster: $CLUSTER_NAME"
echo

# Check if cluster exists
if ! kubectl cluster-info &> /dev/null; then
    echo "ERROR: Cannot connect to Kubernetes cluster"
    echo "Please ensure your cluster is running and kubectl is configured"
    exit 1
fi

# Create namespace
echo "1. Creating namespace..."
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

# Add Helm repo
echo
echo "2. Setting up Helm repositories..."
helm repo add grafana https://grafana.github.io/helm-charts || true
helm repo update

# Deploy infrastructure
echo
echo "3. Deploying Tempo..."
helm upgrade --install tempo grafana/tempo \
    -n $NAMESPACE \
    -f infrastructure/helm/tempo/values.yaml \
    --timeout 300s \
    --wait

echo
echo "4. Deploying Loki..."
helm upgrade --install loki grafana/loki \
    -n $NAMESPACE \
    -f infrastructure/helm/loki/values.yaml \
    --timeout 300s \
    --wait

echo
echo "5. Deploying Mimir..."
helm upgrade --install mimir grafana/mimir-distributed \
    -n $NAMESPACE \
    -f infrastructure/helm/mimir/values.yaml \
    --timeout 300s \
    --wait

echo
echo "6. Deploying Grafana..."
helm upgrade --install grafana grafana/grafana \
    -n $NAMESPACE \
    -f infrastructure/helm/grafana/values.yaml \
    --timeout 300s \
    --wait

# Deploy OTEL Collector
echo
echo "7. Deploying OpenTelemetry Collector..."
kubectl apply -f infrastructure/kubernetes/otel-collector/deployment.yaml

# Build and deploy demo app
echo
echo "8. Building demo application..."
docker build -t demo-app:latest apps/demo-app/

echo
echo "9. Loading demo app into Kind cluster..."
kind load docker-image demo-app:latest --name $CLUSTER_NAME

echo
echo "10. Deploying demo application..."
kubectl apply -f infrastructure/kubernetes/demo-app/deployment.yaml

# Wait for pods
echo
echo "11. Waiting for pods to be ready..."
kubectl wait --for=condition=ready pod -l app=otel-collector -n $NAMESPACE --timeout=120s || true
kubectl wait --for=condition=ready pod -l app=demo-app -n $NAMESPACE --timeout=120s || true

echo
echo "========================================="
echo "✓ Deployment Complete!"
echo "========================================="
echo
echo "Next steps:"
echo "1. Port-forward Grafana: make port-forward"
echo "   Or: kubectl port-forward -n $NAMESPACE svc/grafana 3000:80"
echo
echo "2. Access Grafana: http://localhost:3000"
echo "   Username: admin"
echo "   Password: admin"
echo
echo "3. Generate traffic: make traffic"
echo "   Or: kubectl apply -f infrastructure/kubernetes/traffic-generator/job.yaml"
echo
echo "4. Verify deployment: make verify"
echo "   Or: ./scripts/verify.sh"
echo
