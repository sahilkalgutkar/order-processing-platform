variable "service_name" {
  type = string
}

variable "cluster_arn" {
  type = string
}

variable "image" {
  description = "Full ECR image URI, e.g. <account>.dkr.ecr.<region>.amazonaws.com/order-service:<tag>"
  type        = string
}

variable "container_port" {
  type = number
}

variable "cpu" {
  type    = number
  default = 256
}

variable "memory" {
  type    = number
  default = 512
}

variable "desired_count" {
  type    = number
  default = 2
}

variable "subnet_ids" {
  type = list(string)
}

variable "security_group_ids" {
  type = list(string)
}

variable "environment" {
  description = "Plain environment variables for the container (non-secret config, e.g. AWS_REGION, ports)"
  type        = map(string)
  default     = {}
}

variable "secrets" {
  description = "Map of container env var name -> Secrets Manager / SSM ARN, for DB URLs, credentials, etc."
  type        = map(string)
  default     = {}
}

variable "target_group_arn" {
  description = "ALB target group to register with, if this service is exposed via a load balancer"
  type        = string
  default     = null
}
