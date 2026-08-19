# Mirrors the local topology created by scripts/localstack-init.sh:
# one SNS topic, one SQS queue per consuming service, each subscribed
# with raw message delivery.

resource "aws_sns_topic" "this" {
  name = var.topic_name
}

resource "aws_sqs_queue" "this" {
  for_each                  = var.subscriber_queues
  name                       = each.key
  visibility_timeout_seconds = 30
  message_retention_seconds  = 1209600 # 14 days
}

resource "aws_sqs_queue_policy" "allow_sns" {
  for_each  = var.subscriber_queues
  queue_url = aws_sqs_queue.this[each.key].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "AllowSNSPublish"
      Effect    = "Allow"
      Principal = { Service = "sns.amazonaws.com" }
      Action    = "sqs:SendMessage"
      Resource  = aws_sqs_queue.this[each.key].arn
      Condition = {
        ArnEquals = { "aws:SourceArn" = aws_sns_topic.this.arn }
      }
    }]
  })
}

resource "aws_sns_topic_subscription" "this" {
  for_each                       = var.subscriber_queues
  topic_arn                      = aws_sns_topic.this.arn
  protocol                       = "sqs"
  endpoint                       = aws_sqs_queue.this[each.key].arn
  raw_message_delivery            = each.value
  depends_on                     = [aws_sqs_queue_policy.allow_sns]
}
