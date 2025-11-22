#!/bin/bash

NAMESPACE="${NAMESPACE:-lgtm}"

echo "========================================="
echo "LGTM Stack Cleanup"
echo "========================================="
echo "Namespace: $NAMESPACE"
echo

read -p "Are you sure you want to delete all resources in namespace '$NAMESPACE'? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cleanup cancelled"
    exit 0
fi

echo
echo "1. Removing applications..."
kubectl delete -f infrastructure/kubernetes/demo-app/deployment.yaml --ignore-not-found=true
kubectl delete -f infrastructure/kubernetes/otel-collector/deployment.yaml --ignore-not-found=true
kubectl delete -f infrastructure/kubernetes/traffic-generator/job.yaml --ignore-not-found=true

echo
echo "2. Uninstalling Helm releases..."
helm uninstall grafana -n $NAMESPACE --ignore-not-found || true
helm uninstall tempo -n $NAMESPACE --ignore-not-found || true
helm uninstall loki -n $NAMESPACE --ignore-not-found || true
helm uninstall mimir -n $NAMESPACE --ignore-not-found || true

echo
echo "3. Waiting for resources to terminate..."
sleep 5

echo
read -p "Do you want to delete the namespace '$NAMESPACE'? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Deleting namespace..."
    kubectl delete namespace $NAMESPACE --ignore-not-found=true
    echo "✓ Namespace deleted"
else
    echo "Namespace preserved"
fi

echo
echo "========================================="
echo "✓ Cleanup Complete!"
echo "========================================="
