output "topic_arn" {
  value = aws_sns_topic.this.arn
}

output "queue_urls" {
  value = { for k, q in aws_sqs_queue.this : k => q.id }
}

output "queue_arns" {
  value = { for k, q in aws_sqs_queue.this : k => q.arn }
}
