# curated

Infrastructure patterns I have applied in production, rebuilt from scratch around dummy apps so anyone can run them. No client or former-employer code, configuration or data is included.

Each folder stands on its own: problem, design, trade-offs, how to run it, and proof.

| # | Pattern | Stack |
|---|---|---|
| [01](01-stateless-monolith/) | Make a stateful monolith stateless on Kubernetes without a rewrite | CodeIgniter 3, Redis, Mountpoint S3 CSI, RustFS, k3d / k3s on EC2, Terraform |
| [02](02-progressive-delivery/) | Canary releases gated by readiness and by error rate from the app's own logs, 12-factor app | Go, Postgres, Envoy Gateway (Gateway API), Argo Rollouts, Loki + Alloy, k3s on EC2 |
