variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "cluster_arn" {
  description = "Existing ECS cluster to deploy into"
  type        = string
}

variable "vpc_subnet_ids" {
  type = list(string)
}

variable "security_group_ids" {
  type = list(string)
}

variable "ecr_account_id" {
  description = "AWS account ID hosting the ECR repos referenced below"
  type        = string
}

variable "image_tag" {
  type    = string
  default = "latest"
}
