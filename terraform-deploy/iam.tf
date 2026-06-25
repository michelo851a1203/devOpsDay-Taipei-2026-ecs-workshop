# ==============================
# ecsTaskExecutionRole
# 對應 main.go：先 LookupRole，找不到才建立 role 並掛 policy
# 這裡用 create_ecs_task_execution_role 變數切換建立 / 沿用既有
# ==============================

# 沿用既有 role（create_ecs_task_execution_role = false 時查詢）
data "aws_iam_role" "existing_execution_role" {
  count = var.create_ecs_task_execution_role ? 0 : 1
  name  = "ecsTaskExecutionRole"
}

# 建立 role（create_ecs_task_execution_role = true 時建立）
resource "aws_iam_role" "ecs_task_execution" {
  count = var.create_ecs_task_execution_role ? 1 : 0
  name  = "ecsTaskExecutionRole"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
        Action = "sts:AssumeRole"
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "ecs_task_execution" {
  count      = var.create_ecs_task_execution_role ? 1 : 0
  role       = aws_iam_role.ecs_task_execution[0].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_role_policy_attachment" "ecs_cloudwatch" {
  count      = var.create_ecs_task_execution_role ? 1 : 0
  role       = aws_iam_role.ecs_task_execution[0].name
  policy_arn = "arn:aws:iam::aws:policy/CloudWatchFullAccess"
}

locals {
  execution_role_arn = var.create_ecs_task_execution_role ? aws_iam_role.ecs_task_execution[0].arn : data.aws_iam_role.existing_execution_role[0].arn
}
