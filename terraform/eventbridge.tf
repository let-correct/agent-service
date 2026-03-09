###############################################################################
# Let Correct — EventBridge Custom Event Bus
###############################################################################

resource "aws_cloudwatch_event_bus" "let_correct" {
  name = "let-correct"
  tags = var.tags
}
