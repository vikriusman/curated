CLUSTER   := stateless-gallery
IMAGE     := stateless-gallery:dev
CSI_CHART := aws-mountpoint-s3-csi-driver/aws-mountpoint-s3-csi-driver
CSI_VER   := 2.8.0
KCTX      := --context k3d-$(CLUSTER)

.PHONY: up cluster image deploy csi wait demo status logs down

## up: cluster + image + deploy + wait until ready (local lab, no cloud cost)
up: cluster image deploy csi wait

cluster:
	k3d cluster list $(CLUSTER) >/dev/null 2>&1 || k3d cluster create --config k3d/cluster.yaml

image: image-only
	k3d image import $(IMAGE) -c $(CLUSTER)

# Manifests first: the aws-secret Secret must exist before the CSI driver pods
# start, because credentials are read into env vars at pod creation.
# The bucket Job is immutable once created, so recreate it on every deploy.
deploy:
	kubectl $(KCTX) -n storage delete job create-bucket --ignore-not-found
	kubectl $(KCTX) apply -k k8s/overlays/local

csi:
	helm repo add aws-mountpoint-s3-csi-driver https://awslabs.github.io/mountpoint-s3-csi-driver >/dev/null 2>&1 || true
	helm repo update aws-mountpoint-s3-csi-driver >/dev/null
	helm --kube-context k3d-$(CLUSTER) upgrade --install aws-mountpoint-s3-csi-driver $(CSI_CHART) \
	  --version $(CSI_VER) --namespace kube-system --wait

wait:
	kubectl $(KCTX) -n storage wait --for=condition=complete job/create-bucket --timeout=180s
	kubectl $(KCTX) -n gallery rollout restart deploy/gallery
	kubectl $(KCTX) -n gallery rollout status deploy/gallery --timeout=300s

## demo: log in once, then prove sessions and files are consistent across pods
demo:
	./scripts/demo.sh http://localhost:8080

status:
	kubectl $(KCTX) -n gallery get pods -o wide
	kubectl $(KCTX) -n mount-s3 get pods -o wide

logs:
	kubectl $(KCTX) -n gallery logs -l app=gallery --tail=50 --prefix

down:
	k3d cluster delete $(CLUSTER)

# ---------------------------------------------------------------------------
# AWS: k3s server + agent(s) on EC2, RustFS in-cluster. Costs money while running.
# Flow: make aws-plan -> review -> make aws-up -> make aws-demo -> make aws-stop / aws-down
# ---------------------------------------------------------------------------
TF_DIR      := terraform/aws
# Override with your own profile: AWS_PROFILE=<name> make aws-...
AWS_PROFILE ?= default
K3S_VERSION := v1.35.5+k3s1
AWS_KCFG    := $(TF_DIR)/kubeconfig
AWS_K       := kubectl --kubeconfig $(AWS_KCFG)
SSH_OPTS    := -i $(TF_DIR)/.ssh/id_ed25519 -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=$(TF_DIR)/.ssh/known_hosts -o ConnectTimeout=5
AWS_CLI     := aws --profile $(AWS_PROFILE) --region ap-southeast-1
# Looked up live (not from Terraform state): public IPs change after stop/start.
EC2_Q        = $(AWS_CLI) ec2 describe-instances --output text --filters \
               Name=instance-state-name,Values=pending,running,stopping,stopped
SERVER_Q     = $(EC2_Q) Name=tag:Name,Values=stateless-gallery
PUBLIC_IP    = $$($(SERVER_Q) --query 'Reservations[0].Instances[0].PublicIpAddress')
SERVER_PRIV  = $$($(SERVER_Q) --query 'Reservations[0].Instances[0].PrivateIpAddress')
AGENT_IPS    = $$($(EC2_Q) Name=tag:Name,Values='stateless-gallery-agent-*' --query 'Reservations[].Instances[].PublicIpAddress')
ALL_IDS      = $$($(EC2_Q) Name=tag:Project,Values=curated-01-stateless-monolith --query 'Reservations[].Instances[].InstanceId')

.PHONY: image-only aws-stop aws-start aws-plan aws-up aws-infra aws-kubeconfig aws-join aws-image aws-deploy aws-csi aws-wait aws-demo aws-status aws-down

## aws-plan: write a plan for review (pins SSH/API/HTTP access to your current IP)
aws-plan:
	terraform -chdir=$(TF_DIR) init -input=false >/dev/null
	terraform -chdir=$(TF_DIR) plan -input=false -out=tfplan \
	  -var aws_profile=$(AWS_PROFILE) -var k3s_version=$(K3S_VERSION) \
	  -var allowed_cidr=$$(curl -s https://checkip.amazonaws.com)/32

## aws-up: apply the reviewed plan, join agents, then deploy the same image and manifests
aws-up: aws-infra aws-kubeconfig aws-join image-only aws-image aws-deploy aws-csi aws-wait

aws-infra:
	terraform -chdir=$(TF_DIR) apply -input=false tfplan
	rm -f $(TF_DIR)/tfplan

aws-kubeconfig:
	@IP=$(PUBLIC_IP); \
	echo "waiting for k3s server on $$IP"; \
	for i in $$(seq 60); do ssh $(SSH_OPTS) ubuntu@$$IP test -f /var/lib/k3s-ready 2>/dev/null && break; sleep 5; done; \
	ssh $(SSH_OPTS) ubuntu@$$IP sudo cat /etc/rancher/k3s/k3s.yaml | sed "s#https://127.0.0.1:6443#https://$$IP:6443#" > $(AWS_KCFG); \
	chmod 600 $(AWS_KCFG); \
	$(AWS_K) config set-cluster default --tls-server-name=kubernetes >/dev/null; \
	echo "waiting for the k3s API"; \
	for i in $$(seq 60); do $(AWS_K) get --raw /readyz >/dev/null 2>&1 && break; sleep 5; done; \
	$(AWS_K) get nodes

# Join every agent to the server over the private network. The token travels
# over SSH stdin only: never in user_data, Terraform state or a process list.
# Idempotent: agents that already joined are skipped.
aws-join:
	@SERVER=$(PUBLIC_IP); PRIV=$(SERVER_PRIV); \
	TOKEN=$$(ssh $(SSH_OPTS) ubuntu@$$SERVER sudo cat /var/lib/rancher/k3s/server/node-token); \
	for IP in $(AGENT_IPS); do \
	  echo "joining agent $$IP -> https://$$PRIV:6443"; \
	  for i in $$(seq 60); do ssh $(SSH_OPTS) ubuntu@$$IP true 2>/dev/null && break; sleep 5; done; \
	  printf 'K3S_TOKEN=%s\n' "$$TOKEN" | ssh $(SSH_OPTS) ubuntu@$$IP \
	    "sudo bash -c 'test -f /var/lib/k3s-ready && exit 0; read -r line; export \$$line; \
	     curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=$(K3S_VERSION) K3S_URL=https://$$PRIV:6443 sh -s - agent && touch /var/lib/k3s-ready'"; \
	done
	$(AWS_K) wait --for=condition=ready node --all --timeout=300s
	$(AWS_K) get nodes -o wide

image-only:
	docker build -t $(IMAGE) -f docker/Dockerfile .

# No registry needed: stream the image straight into each node's containerd.
aws-image:
	@for IP in $(PUBLIC_IP) $(AGENT_IPS); do \
	  echo "importing $(IMAGE) into $$IP"; \
	  docker save $(IMAGE) | ssh $(SSH_OPTS) ubuntu@$$IP sudo k3s ctr images import - >/dev/null; \
	done

aws-deploy:
	@printf 'BASE_URL=http://%s/\nSESSION_REDIS_URL=tcp://redis:6379?prefix=gallery:sess:\nUPLOAD_PATH=/data/uploads\n' \
	  "$(PUBLIC_IP)" > k8s/overlays/aws/aws.env
	@test -f k8s/overlays/aws/secret.env || printf 'APP_KEY=%s\nDEMO_PASSWORD=%s\n' \
	  "$$(openssl rand -hex 32)" "$$(openssl rand -base64 18 | tr -d '/+=')" > k8s/overlays/aws/secret.env
	@# One random RustFS key pair, written twice: for RustFS itself and for the CSI driver.
	@test -f k8s/overlays/aws/rustfs.env || { K=$$(openssl rand -hex 10); S=$$(openssl rand -hex 24); \
	  printf 'RUSTFS_ACCESS_KEY=%s\nRUSTFS_SECRET_KEY=%s\n' $$K $$S > k8s/overlays/aws/rustfs.env; \
	  printf 'key_id=%s\naccess_key=%s\n' $$K $$S > k8s/overlays/aws/csi.env; }
	$(AWS_K) -n storage delete job create-bucket --ignore-not-found
	$(AWS_K) apply -k k8s/overlays/aws

aws-csi:
	helm repo add aws-mountpoint-s3-csi-driver https://awslabs.github.io/mountpoint-s3-csi-driver >/dev/null 2>&1 || true
	helm repo update aws-mountpoint-s3-csi-driver >/dev/null
	helm --kubeconfig $(AWS_KCFG) upgrade --install aws-mountpoint-s3-csi-driver $(CSI_CHART) \
	  --version $(CSI_VER) --namespace kube-system --wait

aws-wait:
	$(AWS_K) -n storage wait --for=condition=complete job/create-bucket --timeout=300s
	$(AWS_K) -n gallery rollout restart deploy/gallery
	$(AWS_K) -n gallery rollout status deploy/gallery --timeout=300s

aws-demo:
	set -a; . k8s/overlays/aws/secret.env; set +a; \
	KUBECTL_ARGS="--kubeconfig $(AWS_KCFG)" ./scripts/demo.sh http://$(PUBLIC_IP)

aws-status:
	$(AWS_K) get nodes -o wide
	$(AWS_K) -n gallery get pods -o wide
	$(AWS_K) -n mount-s3 get pods -o wide

## aws-stop: stop every instance, keep their disks (EBS only, ~USD 0.05/day per node)
aws-stop:
	$(AWS_CLI) ec2 stop-instances --instance-ids $(ALL_IDS) --query 'StoppingInstances[].[InstanceId,CurrentState.Name]' --output text
	$(AWS_CLI) ec2 wait instance-stopped --instance-ids $(ALL_IDS)
	@echo stopped

## aws-start: start everything again; new public IPs, so refresh kubeconfig and BASE_URL
## (private IPs survive stop/start, so agents rejoin the server on their own)
aws-start:
	$(AWS_CLI) ec2 start-instances --instance-ids $(ALL_IDS) --query 'StartingInstances[].[InstanceId,CurrentState.Name]' --output text
	$(AWS_CLI) ec2 wait instance-running --instance-ids $(ALL_IDS)
	rm -f $(TF_DIR)/.ssh/known_hosts
	$(MAKE) aws-kubeconfig
	$(AWS_K) wait --for=condition=ready node --all --timeout=300s
	$(MAKE) aws-deploy aws-wait
	@echo "app: http://$(PUBLIC_IP)/"

## aws-down: destroy everything (RustFS data lives on the server disk and goes with it)
aws-down:
	terraform -chdir=$(TF_DIR) destroy -input=false -auto-approve \
	  -var aws_profile=$(AWS_PROFILE) -var allowed_cidr=0.0.0.0/32
	rm -f $(AWS_KCFG) k8s/overlays/aws/aws.env
