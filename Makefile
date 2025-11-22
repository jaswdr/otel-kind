.PHONY: help create-cluster delete-cluster deploy clean status logs verify port-forward

NAMESPACE_LGTM ?= lgtm
NAMESPACE_APP ?= default
CLUSTER_NAME ?= otel-cluster
HELM_TIMEOUT ?= 300s

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Cluster
create-cluster: ## Create Kind cluster
	kind create cluster --name $(CLUSTER_NAME) --config infra/kind-cluster.yaml || true
	kubectl cluster-info --context kind-$(CLUSTER_NAME)

delete-cluster: ## Delete Kind cluster
	kind delete cluster --name $(CLUSTER_NAME)

# Deploy
deploy: create-namespaces deploy-infra deploy-apps ## Deploy everything
	@echo "Deployment complete. Run 'make port-forward' to access Grafana"

create-namespaces:
	kubectl create namespace $(NAMESPACE_LGTM) --dry-run=client -o yaml | kubectl apply -f -

deploy-infra: ## Deploy LGTM stack
	helm repo add grafana https://grafana.github.io/helm-charts || true
	helm repo update
	helm upgrade --install tempo grafana/tempo -n $(NAMESPACE_LGTM) -f infra/helm/tempo-values.yaml --timeout $(HELM_TIMEOUT) --wait
	helm upgrade --install loki grafana/loki -n $(NAMESPACE_LGTM) -f infra/helm/loki-values.yaml --timeout $(HELM_TIMEOUT) --wait
	helm upgrade --install mimir grafana/mimir-distributed -n $(NAMESPACE_LGTM) -f infra/helm/mimir-values.yaml --timeout $(HELM_TIMEOUT) --wait
	helm upgrade --install grafana grafana/grafana -n $(NAMESPACE_LGTM) -f infra/helm/grafana-values.yaml --timeout $(HELM_TIMEOUT) --wait
	kubectl apply -f infra/k8s/01-otel-collector.yaml
	kubectl apply -f infra/k8s/05-promtail.yaml

deploy-apps: build-app load-app ## Deploy demo app and postgres
	kubectl apply -f infra/k8s/02-postgres.yaml
	kubectl wait --for=condition=ready pod -l app=postgres -n $(NAMESPACE_APP) --timeout=120s || true
	kubectl apply -f infra/k8s/03-demo-app.yaml
	kubectl wait --for=condition=ready pod -l app=demo-app -n $(NAMESPACE_APP) --timeout=120s || true
	kubectl apply -f infra/k8s/04-load-generator.yaml

build-app: ## Build demo app
	cd apps/demo-app && docker build -t demo-app:latest .

load-app: ## Load app into cluster
	kind load docker-image demo-app:latest --name $(CLUSTER_NAME)

rebuild-app: build-app load-app ## Rebuild and reload app
	kubectl rollout restart deployment/demo-app -n $(NAMESPACE_APP)

# Utilities
port-forward: ## Port-forward to Grafana
	@echo "Grafana: http://localhost:3000 (admin/admin)"
	kubectl port-forward -n $(NAMESPACE_LGTM) svc/grafana 3000:80

verify: ## Verify deployment
	@./hacks/verify.sh

status: ## Show pod status
	@echo "LGTM namespace:"
	@kubectl get pods -n $(NAMESPACE_LGTM)
	@echo ""
	@echo "App namespace:"
	@kubectl get pods -n $(NAMESPACE_APP)

logs: ## Show demo app logs
	kubectl logs -n $(NAMESPACE_APP) -l app=demo-app --tail=50 -f

# Cleanup
clean: ## Remove everything except namespaces
	kubectl delete -f infra/k8s/03-demo-app.yaml --ignore-not-found=true
	kubectl delete -f infra/k8s/02-postgres.yaml --ignore-not-found=true
	kubectl delete -f infra/k8s/04-load-generator.yaml --ignore-not-found=true
	kubectl delete -f infra/k8s/01-otel-collector.yaml --ignore-not-found=true
	kubectl delete -f infra/k8s/05-promtail.yaml --ignore-not-found=true
	helm uninstall grafana -n $(NAMESPACE_LGTM) --ignore-not-found || true
	helm uninstall tempo -n $(NAMESPACE_LGTM) --ignore-not-found || true
	helm uninstall loki -n $(NAMESPACE_LGTM) --ignore-not-found || true
	helm uninstall mimir -n $(NAMESPACE_LGTM) --ignore-not-found || true
