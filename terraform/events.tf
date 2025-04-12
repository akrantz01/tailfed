resource "aws_cloudwatch_event_bus" "events" {
  name = "tailfed"
}

resource "aws_scheduler_schedule_group" "tailfed" {
  name = "tailfed"
}

resource "aws_scheduler_schedule" "generator" {
  name       = "generator"
  group_name = aws_scheduler_schedule_group.tailfed.name

  state               = "ENABLED"
  schedule_expression = "rate(1 days)"

  flexible_time_window {
    mode                      = "FLEXIBLE"
    maximum_window_in_minutes = 15
  }

  target {
    arn      = module.generator.arn
    role_arn = aws_iam_role.generator_schedule.arn

    input = local.generator_input

    retry_policy {
      maximum_event_age_in_seconds = 3600
      maximum_retry_attempts       = 5
    }
  }
}

resource "aws_iam_role" "generator_schedule" {
  name = "TailfedScheduleInvokeGenerator"
  path = "/tailfed/"

  assume_role_policy = data.aws_iam_policy_document.generator_schedule_trust_policy.json
}

resource "aws_iam_role_policy" "generator_schedule" {
  role   = aws_iam_role.generator_schedule.id
  policy = data.aws_iam_policy_document.generator_schedule.json
}
