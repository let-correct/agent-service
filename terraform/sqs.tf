###############################################################################
# Calendar Sync Worker — SQS Queue + Dead-Letter Queue
###############################################################################

resource "aws_sqs_queue" "calendar_sync_worker_dlq" {
  name              = "calendar-sync-worker-dlq"

  tags = var.tags
}

resource "aws_sqs_queue" "calendar_sync_worker" {
  name              = "calendar-sync-worker"

  visibility_timeout_seconds = 60 # Must be >= Lambda timeout (30s), recommended 2x

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.calendar_sync_worker_dlq.arn
    maxReceiveCount     = 3
  })

  tags = var.tags
}
