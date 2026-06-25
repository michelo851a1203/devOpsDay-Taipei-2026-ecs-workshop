# ==============================
# 對應 main.go 最上方那一段需要自己填的設定
# ==============================

variable "region" {
  description = "AWS region（對應 pulumi config set aws:region）"
  type        = string
  default     = "ap-east-2"
}

variable "aws_profile" {
  description = "AWS profile（對應 pulumi config set aws:profile）。留空字串則使用預設 / 環境變數認證"
  type        = string
  default     = ""
}

variable "vpc_id" {
  description = "使用的 VPC ID，例如 vpc-034121b108f7b40f5"
  type        = string
}

variable "subnet_ids" {
  description = "ALB 與 ECS service 使用的 subnet 清單（對應 main.go 的 subnetA/B/C）"
  type        = list(string)
}

variable "ecr_uri" {
  description = "容器 image，例如 <account_id>.dkr.ecr.ap-east-2.amazonaws.com/<repo>:latest"
  type        = string
}

variable "cpu_architecture" {
  description = "Task 的 CPU 架構：ARM64 或 X86_64"
  type        = string
  default     = "ARM64"
}

variable "blue_service_name" {
  description = "Blue ECS service 名稱"
  type        = string
  default     = "api-blue-service-9ea7cd8ca8e752b2"
}

variable "green_service_name" {
  description = "Green ECS service 名稱"
  type        = string
  default     = "api-green-service-2f43e8d51a80e99b"
}

variable "create_ecs_task_execution_role" {
  description = <<-EOT
    是否建立 ecsTaskExecutionRole。
    對應 main.go 的「先 LookupRole，找不到才建立」邏輯：
      - true ：本模組負責建立 role（IAM 中尚未存在時）
      - false：沿用帳號內既有的 ecsTaskExecutionRole（用 data source 查詢）
  EOT
  type        = bool
  default     = true
}
