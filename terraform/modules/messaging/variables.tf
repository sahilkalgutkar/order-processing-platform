variable "topic_name" {
  description = "Name of the SNS topic events are published to"
  type        = string
}

variable "subscriber_queues" {
  description = "Map of SQS queue name -> whether raw message delivery is enabled"
  type        = map(bool)
}
