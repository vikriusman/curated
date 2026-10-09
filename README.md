# curated

Infrastructure patterns I have applied in production, rebuilt from scratch around dummy apps so anyone can run them. No client or former-employer code, configuration or data is included.

Each folder stands on its own: problem, design, trade-offs, how to run it, and proof.

| # | Pattern | Stack |
|---|---|---|
| [01](01-stateless-monolith/) | Make a stateful monolith stateless on Kubernetes without a rewrite | CodeIgniter 3, Redis, Mountpoint S3 CSI, RustFS, k3d / k3s on EC2, Terraform |
| [02](02-progressive-delivery/) | Canary releases gated by readiness and by error rate from the app's own logs, 12-factor app | Go, Postgres, Envoy Gateway (Gateway API), Argo Rollouts, Loki + Alloy, k3s on EC2 |

## Field notes

Real production work that cannot be reproduced publicly: anonymized, numbers approximate, written to show the constraints, the decisions and what actually happened.

| # | Note | Topics |
|---|---|---|
| [01](field-notes/01-ha-kubernetes-three-regions/) | Highly available on-prem Kubernetes across three regions | etcd and control-plane HA, keepalived + MetalLB, k3s to RKE2 without downtime, power-cut drill, entry point across regions |
| [02](field-notes/02-social-network-outgrew-estimate/) | An internal social network that outgrew its estimate in two months (postmortem) | Capacity estimation, search on growing data, pagination, media out of the database, cost of overconfidence |
