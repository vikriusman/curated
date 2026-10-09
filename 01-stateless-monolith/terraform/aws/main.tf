data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
  filter {
    name   = "default-for-az"
    values = ["true"]
  }
}

data "aws_ssm_parameter" "ubuntu" {
  name = "/aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp3/ami-id"
}

locals {
  name = "stateless-gallery"
}

# Object storage is RustFS inside the cluster (k8s/components/rustfs), so this
# stack needs no S3 bucket and no IAM role: the instance holds no AWS credentials.

# ---------- Network access ----------

resource "aws_security_group" "node" {
  name        = "${local.name}-node"
  description = "k3s single node for the stateless gallery lab"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "SSH (image import, kubeconfig)"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.allowed_cidr]
  }
  ingress {
    description = "k3s API"
    from_port   = 6443
    to_port     = 6443
    protocol    = "tcp"
    cidr_blocks = [var.allowed_cidr]
  }
  ingress {
    description = "HTTP (Traefik ingress)"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = [var.allowed_cidr]
  }
  ingress {
    description = "HTTP (Envoy Gateway, curated/02)"
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = [var.allowed_cidr]
  }
  ingress {
    description = "Node to node: k3s API, kubelet, flannel VXLAN"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    self        = true
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "tls_private_key" "ssh" {
  algorithm = "ED25519"
}

resource "aws_key_pair" "node" {
  key_name   = local.name
  public_key = tls_private_key.ssh.public_key_openssh
}

resource "local_sensitive_file" "ssh_key" {
  filename        = "${path.module}/.ssh/id_ed25519"
  content         = tls_private_key.ssh.private_key_openssh
  file_permission = "0600"
}

# ---------- EC2 + k3s ----------

resource "aws_instance" "node" {
  ami                    = data.aws_ssm_parameter.ubuntu.value
  instance_type          = var.instance_type
  subnet_id              = data.aws_subnets.default.ids[0]
  vpc_security_group_ids = [aws_security_group.node.id]
  key_name               = aws_key_pair.node.key_name

  associate_public_ip_address = true

  # IMDSv2 only, hop limit 1: pods cannot reach instance metadata.
  metadata_options {
    http_tokens                 = "required"
    http_put_response_hop_limit = 1
  }

  root_block_device {
    volume_type = "gp3"
    volume_size = 20
    encrypted   = true
  }

  user_data = templatefile("${path.module}/user_data.sh.tftpl", {
    k3s_version = var.k3s_version
  })

  tags = {
    Name = local.name
  }

  # - associate_public_ip_address: a stopped instance reports none, so every plan
  #   made while the lab is paused (make aws-stop) would replace the instance.
  # - ami: the SSM "current" Ubuntu AMI moves with every Canonical release; picking
  #   it up must be a deliberate rebuild, not a side effect of an unrelated change.
  lifecycle {
    ignore_changes = [associate_public_ip_address, ami]
  }
}

# ---------- k3s agents ----------
# Plain Ubuntu instances. They join the server over SSH (`make aws-join`), so the
# k3s join token never lands in user_data or Terraform state.

resource "aws_instance" "agent" {
  count = var.agent_count

  ami                    = data.aws_ssm_parameter.ubuntu.value
  instance_type          = var.instance_type
  subnet_id              = data.aws_subnets.default.ids[0]
  vpc_security_group_ids = [aws_security_group.node.id]
  key_name               = aws_key_pair.node.key_name

  associate_public_ip_address = true

  metadata_options {
    http_tokens                 = "required"
    http_put_response_hop_limit = 1
  }

  root_block_device {
    volume_type = "gp3"
    volume_size = 20
    encrypted   = true
  }

  tags = {
    Name = "${local.name}-agent-${count.index}"
  }

  # - associate_public_ip_address: a stopped instance reports none, so every plan
  #   made while the lab is paused (make aws-stop) would replace the instance.
  # - ami: the SSM "current" Ubuntu AMI moves with every Canonical release; picking
  #   it up must be a deliberate rebuild, not a side effect of an unrelated change.
  lifecycle {
    ignore_changes = [associate_public_ip_address, ami]
  }
}
