#!/bin/sh
# Runs automatically inside the localstack container on startup
# (mounted to /etc/localstack/init/ready.d/). Creates the SNS topic and
# the two SQS queues (one per consumer service), and subscribes both
# queues with RawMessageDelivery=true so the SQS message body is the
# OrderCreated JSON payload directly, not an SNS notification envelope.
set -e

ENDPOINT="http://localhost:4566"
REGION="us-east-1"

echo "Creating SNS topic order-events..."
TOPIC_ARN=$(awslocal sns create-topic --name order-events --region "$REGION" --query 'TopicArn' --output text)
echo "Topic ARN: $TOPIC_ARN"

echo "Creating SQS queue inventory-queue..."
INVENTORY_QUEUE_URL=$(awslocal sqs create-queue --queue-name inventory-queue --region "$REGION" --query 'QueueUrl' --output text)
INVENTORY_QUEUE_ARN=$(awslocal sqs get-queue-attributes --queue-url "$INVENTORY_QUEUE_URL" --attribute-names QueueArn --region "$REGION" --query 'Attributes.QueueArn' --output text)

echo "Creating SQS queue notification-queue..."
NOTIFICATION_QUEUE_URL=$(awslocal sqs create-queue --queue-name notification-queue --region "$REGION" --query 'QueueUrl' --output text)
NOTIFICATION_QUEUE_ARN=$(awslocal sqs get-queue-attributes --queue-url "$NOTIFICATION_QUEUE_URL" --attribute-names QueueArn --region "$REGION" --query 'Attributes.QueueArn' --output text)

echo "Subscribing inventory-queue to order-events (raw delivery)..."
awslocal sns subscribe \
  --topic-arn "$TOPIC_ARN" \
  --protocol sqs \
  --notification-endpoint "$INVENTORY_QUEUE_ARN" \
  --attributes RawMessageDelivery=true \
  --region "$REGION"

echo "Subscribing notification-queue to order-events (raw delivery)..."
awslocal sns subscribe \
  --topic-arn "$TOPIC_ARN" \
  --protocol sqs \
  --notification-endpoint "$NOTIFICATION_QUEUE_ARN" \
  --attributes RawMessageDelivery=true \
  --region "$REGION"

echo "localstack-init done."
echo "TOPIC_ARN=$TOPIC_ARN"
echo "INVENTORY_QUEUE_URL=$INVENTORY_QUEUE_URL"
echo "NOTIFICATION_QUEUE_URL=$NOTIFICATION_QUEUE_URL"
