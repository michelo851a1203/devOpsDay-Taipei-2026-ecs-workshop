# ==============================
# Security Group（對應 ec2.NewSecurityGroup "demo-sg"）
# ==============================
resource "aws_security_group" "demo" {
  name        = "demo-sg"
  description = "demo-sg description"
  vpc_id      = var.vpc_id

  # 80 對外開放
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # 8080 僅允許同 SG 內互通（對應 Self: true）
  ingress {
    from_port = 8080
    to_port   = 8080
    protocol  = "tcp"
    self      = true
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# ==============================
# Application Load Balancer（對應 alb.NewLoadBalancer "demo-alb"）
# ==============================
resource "aws_lb" "demo" {
  name               = "demo-alb"
  load_balancer_type = "application"
  internal           = false
  security_groups    = [aws_security_group.demo.id]
  subnets            = var.subnet_ids
}

# ==============================
# Target Group（對應 alb.NewTargetGroup "demo-tg"）
# ==============================
resource "aws_lb_target_group" "demo" {
  name        = "demo-tg"
  target_type = "ip"
  protocol    = "HTTP"
  port        = 8080
  vpc_id      = var.vpc_id

  health_check {
    path                = "/health"
    port                = "8080"
    healthy_threshold   = 2
    unhealthy_threshold = 3
    interval            = 10
  }
}

# ==============================
# ALB Listener（對應 lb.NewListener "alb-listener"）
# ==============================
resource "aws_lb_listener" "demo" {
  load_balancer_arn = aws_lb.demo.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.demo.arn
  }
}
