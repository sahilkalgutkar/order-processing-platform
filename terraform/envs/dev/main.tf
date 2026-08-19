terraform {
  required_version = ">= 1.7"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

module "messaging" {
  source     = "../../modules/messaging"
  topic_name = "order-events"
  subscriber_queues = {
    "inventory-queue"     = true
    "notification-queue"  = true
  }
}

module "order_service" {
  source         = "../../modules/ecs-service"
  service_name   = "order-service"
  cluster_arn    = var.cluster_arn
  image          = "${var.ecr_account_id}.dkr.ecr.${var.aws_region}.amazonaws.com/order-service:${var.image_tag}"
  container_port = 8080
  subnet_ids     = var.vpc_subnet_ids
  security_group_ids = var.security_group_ids

  environment = {
    HTTP_PORT     = "8080"
    AWS_REGION    = var.aws_region
    SNS_TOPIC_ARN = module.messaging.topic_arn
  }

  secrets = {
    DATABASE_URL = "arn:aws:secretsmanager:${var.aws_region}:${var.ecr_account_id}:secret:order-service/database-url"
  }
}

module "inventory_service" {
  source         = "../../modules/ecs-service"
  service_name   = "inventory-service"
  cluster_arn    = var.cluster_arn
  image          = "${var.ecr_account_id}.dkr.ecr.${var.aws_region}.amazonaws.com/inventory-service:${var.image_tag}"
  container_port = 8081
  subnet_ids     = var.vpc_subnet_ids
  security_group_ids = var.security_group_ids

  environment = {
    HTTP_PORT    = "8081"
    AWS_REGION   = var.aws_region
    SQS_QUEUE_URL = module.messaging.queue_urls["inventory-queue"]
  }

  secrets = {
    MONGO_URI = "arn:aws:secretsmanager:${var.aws_region}:${var.ecr_account_id}:secret:inventory-service/mongo-uri"
  }
}

module "notification_service" {
  source         = "../../modules/ecs-service"
  service_name   = "notification-service"
  cluster_arn    = var.cluster_arn
  image          = "${var.ecr_account_id}.dkr.ecr.${var.aws_region}.amazonaws.com/notification-service:${var.image_tag}"
  container_port = 8082
  subnet_ids     = var.vpc_subnet_ids
  security_group_ids = var.security_group_ids

  environment = {
    HTTP_PORT     = "8082"
    AWS_REGION    = var.aws_region
    SQS_QUEUE_URL = module.messaging.queue_urls["notification-queue"]
  }
}

# Least-privilege IAM per service, attached to each task role rather than
# baked into the ecs-service module: order-service only ever publishes,
# the two consumers only ever receive/delete on their own queue.

resource "aws_iam_role_policy" "order_service_sns_publish" {
  name = "sns-publish"
  role = module.order_service.task_role_name
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["sns:Publish"]
      Resource = module.messaging.topic_arn
    }]
  })
}

resource "aws_iam_role_policy" "inventory_service_sqs_consume" {
  name = "sqs-consume"
  role = module.inventory_service.task_role_name
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["sqs:ReceiveMessage", "sqs:DeleteMessage", "sqs:GetQueueAttributes"]
      Resource = module.messaging.queue_arns["inventory-queue"]
    }]
  })
}

resource "aws_iam_role_policy" "notification_service_sqs_consume" {
  name = "sqs-consume"
  role = module.notification_service.task_role_name
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["sqs:ReceiveMessage", "sqs:DeleteMessage", "sqs:GetQueueAttributes"]
      Resource = module.messaging.queue_arns["notification-queue"]
    }]
  })
}
