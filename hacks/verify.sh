#!/bin/bash

echo "========================================="
echo "LGTM Stack Verification Script"
echo "========================================="
echo

# Check if port-forward is running
if ! pgrep -f "port-forward.*grafana" > /dev/null; then
    echo "Starting Grafana port-forward..."
    kubectl port-forward -n lgtm svc/grafana 3000:80 > /dev/null 2>&1 &
    sleep 3
fi

echo "1. Checking Grafana Access..."
if curl -s -o /dev/null -w "%{http_code}" http://admin:admin@localhost:3000/api/health | grep -q "200"; then
    echo "   ✓ Grafana is accessible"
else
    echo "   ✗ Grafana is not accessible"
    exit 1
fi
echo

echo "2. Checking Data Sources..."
echo "   Data sources configured in Grafana:"
curl -s http://admin:admin@localhost:3000/api/datasources | jq -r '.[] | select(.name=="Loki" or .name=="Mimir" or .name=="Tempo") | "   - \(.name) (\(.type)): \(.url)"'
echo

echo "3. Checking Pod Status..."
echo "   Key pods in lgtm namespace:"
kubectl get pods -n lgtm | grep -E "(demo-app|otel-collector|grafana|tempo-0|loki-0|mimir-gateway)" | awk '{print "   - " $1 ": " $3}'
echo

echo "4. Checking Demo App..."
POD_NAME=$(kubectl get pod -n lgtm -l app=demo-app -o jsonpath='{.items[0].metadata.name}')
if [ -n "$POD_NAME" ]; then
    echo "   ✓ Demo app pod: $POD_NAME"
    LOG_COUNT=$(kubectl logs -n lgtm $POD_NAME --tail=10 | grep -c "Request completed" || echo "0")
    echo "   ✓ Recent requests logged: $LOG_COUNT"
else
    echo "   ✗ Demo app pod not found"
fi
echo

echo "5. Checking OTEL Collector..."
OTEL_POD=$(kubectl get pod -n lgtm -l app=otel-collector -o jsonpath='{.items[0].metadata.name}')
if [ -n "$OTEL_POD" ]; then
    echo "   ✓ OTEL Collector pod: $OTEL_POD"
    TRACE_COUNT=$(kubectl logs -n lgtm $OTEL_POD --tail=100 | grep -c "Span #" || echo "0")
    echo "   ✓ Traces processed: $TRACE_COUNT+ spans"
else
    echo "   ✗ OTEL Collector pod not found"
fi
echo

echo "========================================="
echo "Verification complete!"
echo "========================================="
echo
echo "Next steps:"
echo "1. Open Grafana: http://localhost:3000"
echo "2. Login with admin/admin"
echo "3. Go to Explore and select each data source:"
echo "   - Loki: Query {service_name=\"demo-app\"}"
echo "   - Mimir: Query http_requests_total"
echo "   - Tempo: Search for service 'demo-app'"
echo
echo "To generate more traffic:"
echo "   kubectl apply -f traffic-generator.yaml"
echo
