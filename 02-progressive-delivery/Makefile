# curated/02 runs on the EC2 k3s cluster provisioned by curated/01
# (server + agent, Terraform in ../01-stateless-monolith/terraform/aws).
INFRA       := ../01-stateless-monolith
AWS_PROFILE ?= default
export AWS_PROFILE
KCFG        := $(INFRA)/terraform/aws/kubeconfig
K           := kubectl --kubeconfig $(KCFG)
AR          := kubectl argo rollouts --kubeconfig $(KCFG)   # plugin flags must follow the plugin name
HELM        := helm --kubeconfig $(KCFG)
SECRETS     := .secrets
STABLE      := v1.0.0
SERVER_IP    = $$(aws --region ap-southeast-1 ec2 describe-instances --output text \
               --filters Name=tag:Name,Values=stateless-gallery Name=instance-state-name,Values=running \
               --query 'Reservations[0].Instances[0].PublicIpAddress')

.PHONY: start stop pause-01 platform secrets image deploy up release demo status watch dashboard url test teardown

## start: start the EC2 nodes (via 01), then free their memory by scaling 01's workloads to 0
start:
	$(MAKE) -C $(INFRA) aws-start
	$(MAKE) pause-01

pause-01:
	-$(K) -n gallery scale deploy --all --replicas=0
	-$(K) -n storage scale deploy --all --replicas=0

## stop: stop the EC2 nodes (disks and data kept)
stop:
	$(MAKE) -C $(INFRA) aws-stop

## platform: Envoy Gateway (:8080), Argo Rollouts + Gateway API plugin, Loki + Alloy, analysis template
platform:
	$(HELM) upgrade --install eg oci://docker.io/envoyproxy/gateway-helm --version v1.9.2 \
	  -n envoy-gateway-system --create-namespace -f k8s/platform/envoy-gateway-values.yaml --wait
	$(K) apply -f k8s/platform/gateway.yaml
	$(K) -n gateway wait --for=condition=Programmed gateway/public --timeout=180s
	helm repo add argo https://argoproj.github.io/argo-helm >/dev/null 2>&1 || true
	helm repo update argo >/dev/null
	$(HELM) upgrade --install argo-rollouts argo/argo-rollouts --version 2.43.5 \
	  -n argo-rollouts --create-namespace -f k8s/platform/argo-rollouts-values.yaml --wait
	$(K) apply -f k8s/platform/dashboard-route.yaml
	helm repo add grafana https://grafana.github.io/helm-charts >/dev/null 2>&1 || true
	helm repo update grafana >/dev/null
	$(HELM) upgrade --install loki grafana/loki --version 7.3.0 \
	  -n logging --create-namespace -f k8s/platform/loki-values.yaml --wait --timeout 10m
	$(HELM) upgrade --install alloy grafana/alloy --version 1.13.1 \
	  -n logging -f k8s/platform/alloy-values.yaml --wait
	$(K) apply -f k8s/platform/analysis-loki.yaml

## secrets: random DB credentials (kept in git-ignored .secrets/), applied as Secrets
secrets:
	@mkdir -p $(SECRETS)
	@test -f $(SECRETS)/postgres.env || { P=$$(openssl rand -hex 16); \
	  printf 'POSTGRES_USER=notes\nPOSTGRES_PASSWORD=%s\nPOSTGRES_DB=notes\n' $$P > $(SECRETS)/postgres.env; \
	  printf 'DATABASE_URL=postgres://notes:%s@postgres:5432/notes?sslmode=disable\n' $$P > $(SECRETS)/db.env; }
	$(K) apply -f k8s/app/00-namespace.yaml
	$(K) -n notes create secret generic postgres --from-env-file=$(SECRETS)/postgres.env --dry-run=client -o yaml | $(K) apply -f -
	$(K) -n notes create secret generic notes-db --from-env-file=$(SECRETS)/db.env --dry-run=client -o yaml | $(K) apply -f -

## image: build once per version (version stamped into the binary), import into every node over SSH
image:
	@test -n "$(VERSION)" || { echo "usage: make image VERSION=v1.1.0"; exit 1; }
	docker build -t notes:$(VERSION) --build-arg VERSION=$(VERSION) app
	$(MAKE) -C $(INFRA) aws-image IMAGE=notes:$(VERSION)

# First install: database, then schema (admin process), then the web process.
# Initial state only: re-running it resets the app to v1.0.0 with default config.
deploy: secrets
	$(K) apply -f k8s/app/10-postgres.yaml
	$(K) -n notes rollout status statefulset/postgres --timeout=300s
	sed "s#image: notes:TAG#image: notes:$(STABLE)#" k8s/jobs/migrate.yaml | $(K) apply -f -
	$(K) -n notes wait --for=condition=complete job/notes-migrate --timeout=120s
	$(K) apply -f k8s/app/
	$(AR) -n notes status notes --timeout 300s

## up: start nodes, platform, notes v1.0.0
up: start platform
	$(MAKE) image VERSION=$(STABLE)
	$(MAKE) deploy

## release: make release VERSION=v1.1.0 [CONFIG="FAULT_RATE=0.3 LOG_LEVEL=debug"]
release:
	@test -n "$(VERSION)" || { echo "usage: make release VERSION=v1.1.0 [CONFIG=\"KEY=VALUE ...\"]"; exit 1; }
	$(MAKE) image VERSION=$(VERSION)
	KCTX="--kubeconfig $(KCFG)" scripts/release.sh $(VERSION) $(CONFIG)

## demo: good release promoted, not-Ready release aborted by readiness, faulty release aborted by Loki analysis
demo:
	$(MAKE) image VERSION=v1.1.0
	$(MAKE) image VERSION=v1.2.0
	KCTX="--kubeconfig $(KCFG)" BASE="http://$(SERVER_IP):8080" scripts/demo.sh

status:
	$(AR) -n notes get rollout notes
	$(K) -n notes get httproute notes -o jsonpath='{range .spec.rules[0].backendRefs[*]}{.name}={.weight}{"  "}{end}{"\n"}'

watch:
	$(AR) -n notes get rollout notes --watch

url:
	@echo "http://$(SERVER_IP):8080/"

## dashboard: Argo Rollouts UI, published at http://<server-ip>:8080/rollouts/ (security group: your IP only)
dashboard:
	@echo "http://$(SERVER_IP):8080/rollouts/"

test:
	docker run --rm -u $$(id -u):$$(id -g) -e HOME=/tmp -e GOCACHE=/tmp/gocache -e GOMODCACHE=/tmp/gomod -v $(CURDIR)/app:/src -w /src golang:1.27 go test ./...

## teardown: remove 02 from the cluster (nodes keep running; use `make stop` after)
teardown:
	-$(K) delete namespace notes --ignore-not-found
	-$(K) delete -f k8s/platform/dashboard-route.yaml --ignore-not-found
	-$(K) delete -f k8s/platform/gateway.yaml --ignore-not-found
	-$(K) delete -f k8s/platform/analysis-loki.yaml --ignore-not-found
	-$(HELM) uninstall alloy -n logging
	-$(HELM) uninstall loki -n logging
	-$(HELM) uninstall argo-rollouts -n argo-rollouts
	-$(HELM) uninstall eg -n envoy-gateway-system
