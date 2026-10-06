variable "region" {
  type    = string
  default = "ap-southeast-1"
}

variable "aws_profile" {
  type    = string
  default = null
}

variable "instance_type" {
  description = "k3s server and agent size. t3.small (2 vCPU / 2 GiB) fits the whole stack."
  type        = string
  default     = "t3.small"
}

variable "k3s_version" {
  type    = string
  default = "v1.35.5+k3s1"
}

variable "allowed_cidr" {
  description = "CIDR allowed to reach SSH, the k3s API and HTTP. Use your own /32."
  type        = string
}

variable "agent_count" {
  description = "k3s agent nodes joined to the server. 1+ makes the cross-node proof run on separate machines."
  type        = number
  default     = 1
}
