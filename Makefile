.PHONY: help create-cluster delete-cluster deploy deploy-infra deploy-apps verify clean port-forward build-demo-app traffic status logs-demo logs-otel logs-grafana logs-postgres logs-load-generator

# Variables
NAMESPACE_LGTM ?= lgtm
NAMESPACE_APP ?= default
CLUSTER_NAME ?= otel-cluster
HELM_TIMEOUT ?= 300s

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-25s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Cluster Management
create-cluster: ## Create a new Kind cluster
	@echo "Creating Kind cluster: $(CLUSTER_NAME)..."
	@kind create cluster --name $(CLUSTER_NAME) --config - <<EOF || true\n\
	kind: Cluster\n\
	apiVersion: kind.x-k8s.io/v1alpha4\n\
	nodes:\n\
	- role: control-plane\n\
	- role: worker\n\
	- role: worker\n\
	EOF
	@echo "✓ Cluster $(CLUSTER_NAME) created"
	@kubectl cluster-info --context kind-$(CLUSTER_NAME)

delete-cluster: ## Delete the Kind cluster
	@echo "Deleting Kind cluster: $(CLUSTER_NAME)..."
	@kind delete cluster --name $(CLUSTER_NAME)
	@echo "✓ Cluster deleted"

# Full Deployment
deploy: create-namespaces deploy-infra deploy-postgres deploy-apps deploy-load-generator ## Deploy everything (cluster + infra + apps)
	@echo ""
	@echo "========================================="
	@echo "✓ Full deployment complete!"
	@echo "========================================="
	@echo ""
	@echo "Next steps:"
	@echo "  1. Port-forward Grafana: make port-forward"
	@echo "  2. Access Grafana: http://localhost:3000 (admin/admin)"
	@echo "  3. Check status: make status"
	@echo "  4. View logs: make logs-demo"
	@echo ""

create-namespaces: ## Create required namespaces
	@echo "Creating namespaces..."
	@kubectl create namespace $(NAMESPACE_LGTM) --dry-run=client -o yaml | kubectl apply -f -
	@echo "✓ Namespace $(NAMESPACE_LGTM) ready"
	@echo "✓ Namespace $(NAMESPACE_APP) ready (default)"

# Infrastructure Deployment
deploy-infra: deploy-helm ## Deploy LGTM stack infrastructure
	@echo "✓ Infrastructure deployed to $(NAMESPACE_LGTM) namespace"

deploy-helm: ## Deploy all Helm charts (Grafana, Tempo, Loki, Mimir)
	@echo "Deploying Helm charts to $(NAMESPACE_LGTM) namespace..."
	@helm repo add grafana https://grafana.github.io/helm-charts || true
	@helm repo update
	@echo ""
	@echo "Deploying Tempo..."
	@helm upgrade --install tempo grafana/tempo \
		-n $(NAMESPACE_LGTM) \
		-f infra/helm/tempo-values.yaml \
		--timeout $(HELM_TIMEOUT) \
		--wait
	@echo ""
	@echo "Deploying Loki..."
	@helm upgrade --install loki grafana/loki \
		-n $(NAMESPACE_LGTM) \
		-f infra/helm/loki-values.yaml \
		--timeout $(HELM_TIMEOUT) \
		--wait
	@echo ""
	@echo "Deploying Mimir..."
	@helm upgrade --install mimir grafana/mimir-distributed \
		-n $(NAMESPACE_LGTM) \
		-f infra/helm/mimir-values.yaml \
		--timeout $(HELM_TIMEOUT) \
		--wait
	@echo ""
	@echo "Deploying Grafana..."
	@helm upgrade --install grafana grafana/grafana \
		-n $(NAMESPACE_LGTM) \
		-f infra/helm/grafana-values.yaml \
		--timeout $(HELM_TIMEOUT) \
		--wait
	@echo ""
	@echo "Deploying OTEL Collector..."
	@kubectl apply -f infra/k8s/01-otel-collector.yaml
	@kubectl wait --for=condition=ready pod -l app=otel-collector -n $(NAMESPACE_LGTM) --timeout=120s || true
	@echo "✓ Helm charts deployed"

# Application Deployment
deploy-apps: deploy-postgres deploy-demo-app ## Deploy all applications
	@echo "✓ Applications deployed to $(NAMESPACE_APP) namespace"

deploy-postgres: ## Deploy PostgreSQL database
	@echo "Deploying PostgreSQL to $(NAMESPACE_APP) namespace..."
	@kubectl apply -f infra/k8s/02-postgres.yaml
	@echo "Waiting for PostgreSQL to be ready..."
	@kubectl wait --for=condition=ready pod -l app=postgres -n $(NAMESPACE_APP) --timeout=120s || true
	@echo "✓ PostgreSQL deployed"

deploy-demo-app: build-demo-app load-demo-app ## Build and deploy demo application
	@echo "Deploying demo-app to $(NAMESPACE_APP) namespace..."
	@kubectl apply -f infra/k8s/03-demo-app.yaml
	@kubectl wait --for=condition=ready pod -l app=demo-app -n $(NAMESPACE_APP) --timeout=120s || true
	@echo "✓ Demo app deployed"

deploy-load-generator: ## Deploy continuous load generator
	@echo "Deploying load generator to $(NAMESPACE_APP) namespace..."
	@kubectl apply -f infra/k8s/04-load-generator.yaml
	@kubectl wait --for=condition=ready pod -l app=load-generator -n $(NAMESPACE_APP) --timeout=60s || true
	@echo "✓ Load generator deployed"

# Build targets
build-demo-app: ## Build demo application Docker image
	@echo "Building demo-app..."
	@cd apps/demo-app && docker build -t demo-app:latest .
	@echo "✓ Demo app built"

load-demo-app: ## Load demo app image into Kind cluster
	@echo "Loading demo-app into Kind cluster..."
	@kind load docker-image demo-app:latest --name $(CLUSTER_NAME)
	@echo "✓ Demo app loaded into cluster"

# Utility targets
port-forward: ## Start port forwarding to Grafana
	@echo "Starting port-forward to Grafana on http://localhost:3000"
	@echo "Credentials: admin/admin"
	@kubectl port-forward -n $(NAMESPACE_LGTM) svc/grafana 3000:80

verify: ## Run verification script
	@echo "Running verification..."
	@./hacks/verify.sh

status: ## Show status of all pods
	@echo "Pods in $(NAMESPACE_LGTM) namespace (LGTM Stack):"
	@kubectl get pods -n $(NAMESPACE_LGTM)
	@echo ""
	@echo "Pods in $(NAMESPACE_APP) namespace (Applications):"
	@kubectl get pods -n $(NAMESPACE_APP)

# Logs
logs-demo: ## Show logs from demo app
	@kubectl logs -n $(NAMESPACE_APP) -l app=demo-app --tail=50 -f

logs-otel: ## Show logs from OTEL collector
	@kubectl logs -n $(NAMESPACE_LGTM) -l app=otel-collector --tail=50 -f

logs-grafana: ## Show logs from Grafana
	@kubectl logs -n $(NAMESPACE_LGTM) -l app.kubernetes.io/name=grafana --tail=50 -f

logs-postgres: ## Show logs from PostgreSQL
	@kubectl logs -n $(NAMESPACE_APP) -l app=postgres --tail=50 -f

logs-load-generator: ## Show logs from load generator
	@kubectl logs -n $(NAMESPACE_APP) -l app=load-generator --tail=50 -f

# Cleanup targets
clean: clean-apps clean-infra ## Clean everything except namespaces
	@echo "✓ Cleanup complete"

clean-apps: ## Remove applications from default namespace
	@echo "Removing applications from $(NAMESPACE_APP) namespace..."
	@kubectl delete -f infra/k8s/03-demo-app.yaml --ignore-not-found=true
	@kubectl delete -f infra/k8s/02-postgres.yaml --ignore-not-found=true
	@kubectl delete -f infra/k8s/04-load-generator.yaml --ignore-not-found=true
	@echo "✓ Applications removed"

clean-infra: ## Remove infrastructure from lgtm namespace
	@echo "Removing infrastructure from $(NAMESPACE_LGTM) namespace..."
	@kubectl delete -f infra/k8s/01-otel-collector.yaml --ignore-not-found=true
	@helm uninstall grafana -n $(NAMESPACE_LGTM) --ignore-not-found || true
	@helm uninstall tempo -n $(NAMESPACE_LGTM) --ignore-not-found || true
	@helm uninstall loki -n $(NAMESPACE_LGTM) --ignore-not-found || true
	@helm uninstall mimir -n $(NAMESPACE_LGTM) --ignore-not-found || true
	@echo "✓ Infrastructure removed"

clean-namespace-lgtm: ## Delete the lgtm namespace
	@kubectl delete namespace $(NAMESPACE_LGTM) --ignore-not-found=true
	@echo "✓ Namespace $(NAMESPACE_LGTM) deleted"

# Development targets
restart-demo: ## Restart demo app
	@kubectl rollout restart deployment/demo-app -n $(NAMESPACE_APP)
	@kubectl rollout status deployment/demo-app -n $(NAMESPACE_APP)
	@echo "✓ Demo app restarted"

restart-otel: ## Restart OTEL collector
	@kubectl rollout restart deployment/otel-collector -n $(NAMESPACE_LGTM)
	@kubectl rollout status deployment/otel-collector -n $(NAMESPACE_LGTM)
	@echo "✓ OTEL collector restarted"

restart-load-generator: ## Restart load generator
	@kubectl rollout restart deployment/load-generator -n $(NAMESPACE_APP)
	@kubectl rollout status deployment/load-generator -n $(NAMESPACE_APP)
	@echo "✓ Load generator restarted"

rebuild-demo: build-demo-app load-demo-app restart-demo ## Rebuild and restart demo app
	@echo "✓ Demo app rebuilt and restarted"

# Quick start
quick-start: create-cluster deploy port-forward ## Create cluster, deploy everything, and start port-forward
	@echo ""
	@echo "========================================="
	@echo "Quick start complete!"
	@echo "========================================="
	@echo "Grafana is now accessible at http://localhost:3000"
	@echo "Username: admin"
	@echo "Password: admin"
	@echo ""
