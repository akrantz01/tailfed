resource "aws_iam_role" "handler" {
  name = "Tailfed${title(var.name)}LambdaRole"
  path = "/tailfed/"

  assume_role_policy = data.aws_iam_policy_document.trust_policy.json
}

resource "aws_iam_role_policy" "handler" {
  role   = aws_iam_role.handler.id
  policy = data.aws_iam_policy_document.permissions_policy.json
}

data "aws_iam_policy_document" "permissions_policy" {
  statement {
    sid    = "CloudWatchLogStream"
    effect = "Allow"
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvent",
    ]
    resources = ["${aws_cloudwatch_log_group.handler.arn}:*"]
  }
}

data "aws_iam_policy_document" "trust_policy" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}
