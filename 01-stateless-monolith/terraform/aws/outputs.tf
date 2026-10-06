output "public_ip" {
  value = aws_instance.node.public_ip
}

output "region" {
  value = var.region
}

output "ssh" {
  value = "ssh -i ${local_sensitive_file.ssh_key.filename} ubuntu@${aws_instance.node.public_ip}"
}

output "server_private_ip" {
  value = aws_instance.node.private_ip
}
