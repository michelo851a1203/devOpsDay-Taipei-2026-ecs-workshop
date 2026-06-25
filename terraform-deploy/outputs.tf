# 對應 main.go 最後的 ctx.Export(...)
output "albDnsName" {
  description = "ALB 的 DNS 名稱（瀏覽器打這個）"
  value       = aws_lb.demo.dns_name
}

output "clusterArn" {
  value = aws_ecs_cluster.demo.arn
}

output "clusterName" {
  value = aws_ecs_cluster.demo.name
}

output "blueServiceName" {
  value = var.blue_service_name
}

output "greenServiceName" {
  value = var.green_service_name
}

output "targetGroupArn" {
  value = aws_lb_target_group.demo.arn
}

output "namespaceId" {
  value = aws_service_discovery_http_namespace.workshop.id
}

output "securityGroupId" {
  value = aws_security_group.demo.id
}
