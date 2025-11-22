#!/bin/bash
set -e

echo "Verifying LGTM Stack..."
echo ""

# Check pods
echo "Checking pods in lgtm namespace:"
kubectl get pods -n lgtm
echo ""

echo "Checking pods in default namespace:"
kubectl get pods -n default
echo ""

# Check Grafana accessibility
echo "Checking Grafana (requires port-forward on 3000):"
if curl -sf http://admin:admin@localhost:3000/api/health > /dev/null 2>&1; then
    echo "✓ Grafana is accessible at http://localhost:3000"
else
    echo "✗ Grafana not accessible. Run: make port-forward"
fi
echo ""

echo "To access Grafana:"
echo "  1. Run: make port-forward"
echo "  2. Open: http://localhost:3000"
echo "  3. Login: admin/admin"
