# ==============================
# Service Discovery HTTP Namespace（對應 servicediscovery.NewHttpNamespace "workshop.local"）
# ==============================
resource "aws_service_discovery_http_namespace" "workshop" {
  name = "workshop.local"
}

# ==============================
# ECS Cluster（對應 ecs.NewCluster "demo-cluster"）
# Pulumi 預設用隨機名稱；Terraform 需明確命名，這裡用 random_id 模擬隨機後綴
# ==============================
resource "random_id" "cluster" {
  byte_length = 4
}

resource "aws_ecs_cluster" "demo" {
  name = "demo-cluster-${random_id.cluster.hex}"

  setting {
    name  = "containerInsights"
    value = "enabled"
  }
}

# ==============================
# Container definitions（對應 main.go 的 blueDefs / greenDefs + json.Marshal）
# ==============================
locals {
  blue_container_definitions = jsonencode([
    {
      name      = "api"
      image     = var.ecr_uri
      essential = true
      portMappings = [
        {
          name          = "api-port"
          containerPort = 8080
          protocol      = "tcp"
          appProtocol   = "http"
        }
      ]
      environment = [
        { name = "APP_VERSION", value = "blue" },
        { name = "FAIL_RATE", value = "0" },
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = "/ecs/sample-log"
          "awslogs-region"        = var.region
          "awslogs-stream-prefix" = "blue"
          "awslogs-create-group"  = "true"
        }
      }
    }
  ])

  green_container_definitions = jsonencode([
    {
      name      = "api"
      image     = var.ecr_uri
      essential = true
      portMappings = [
        {
          name          = "api-port"
          containerPort = 8080
          protocol      = "tcp"
          appProtocol   = "http"
        }
      ]
      environment = [
        { name = "APP_VERSION", value = "green" },
        { name = "FAIL_RATE", value = "0" },
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = "/ecs/sample-log"
          "awslogs-region"        = var.region
          "awslogs-stream-prefix" = "green"
          "awslogs-create-group"  = "true"
        }
      }
    }
  ])
}

# ==============================
# Task Definitions（對應 ecs.NewTaskDefinition "api-blue" / "api-green"）
# ==============================
resource "aws_ecs_task_definition" "blue" {
  family                   = "api-blue"
  cpu                      = "512"
  memory                   = "1024"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = local.execution_role_arn
  container_definitions    = local.blue_container_definitions

  runtime_platform {
    operating_system_family = "LINUX"
    cpu_architecture        = var.cpu_architecture
  }
}

resource "aws_ecs_task_definition" "green" {
  family                   = "api-green"
  cpu                      = "512"
  memory                   = "1024"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = local.execution_role_arn
  container_definitions    = local.green_container_definitions

  runtime_platform {
    operating_system_family = "LINUX"
    cpu_architecture        = var.cpu_architecture
  }
}

# ==============================
# ECS Services（對應 ecs.NewService "api-blue-service" / "api-green-service"）
# ==============================
resource "aws_ecs_service" "blue" {
  name            = var.blue_service_name
  cluster         = aws_ecs_cluster.demo.arn
  task_definition = aws_ecs_task_definition.blue.arn
  desired_count   = 2
  launch_type     = "FARGATE"

  network_configuration {
    assign_public_ip = true
    security_groups  = [aws_security_group.demo.id]
    subnets          = var.subnet_ids
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.demo.arn
    container_name   = "api"
    container_port   = 8080
  }

  service_connect_configuration {
    enabled   = true
    namespace = aws_service_discovery_http_namespace.workshop.arn

    service {
      port_name      = "api-port"
      discovery_name = "api-blue"

      client_alias {
        port     = 8080
        dns_name = "api-blue"
      }
    }
  }

  depends_on = [aws_lb_listener.demo]
}

resource "aws_ecs_service" "green" {
  name            = var.green_service_name
  cluster         = aws_ecs_cluster.demo.arn
  task_definition = aws_ecs_task_definition.green.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    assign_public_ip = true
    security_groups  = [aws_security_group.demo.id]
    subnets          = var.subnet_ids
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.demo.arn
    container_name   = "api"
    container_port   = 8080
  }

  service_connect_configuration {
    enabled   = true
    namespace = aws_service_discovery_http_namespace.workshop.arn

    service {
      port_name      = "api-port"
      discovery_name = "api-green"

      client_alias {
        port     = 8080
        dns_name = "api-green"
      }
    }
  }

  depends_on = [aws_lb_listener.demo]
}
