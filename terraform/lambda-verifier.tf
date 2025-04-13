locals {
  state_machine_log_level_conversion = {
    panic = "FATAL"
    fatal = "FATAL"
    error = "ERROR"
    warn  = "ERROR"
    info  = "ERROR"
    debug = "ALL"
    trace = "ALL"
  }
}

module "verifier" {
  source = "./modules/lambda"

  depends_on = [aws_s3_object_copy.artifacts["verifier"]]

  name = "verifier"
  arch = var.architecture

  bucket   = module.artifacts_proxy.id
  checksum = local.artifact_hashes["verifier"]

  timeout = 60

  environment = {
    TAILFED_LOG_LEVEL           = var.log_level
    TAILFED_STORAGE__TABLE      = aws_dynamodb_table.storage.name
    TAILFED_TAILSCALE__TAILNET  = var.tailscale.tailnet
    TAILFED_TAILSCALE__AUTH_KEY = var.tailscale.auth_key
  }
}

resource "aws_iam_role_policy" "verifier" {
  role = module.verifier.role_id

  name   = "Lambda"
  policy = data.aws_iam_policy_document.verifier.json
}

resource "aws_iam_role" "verifier_state_machine" {
  name = "TailfedVerifierStateMachine"
  path = "/tailfed/"

  assume_role_policy = data.aws_iam_policy_document.verifier_state_machine_trust_policy.json
}

resource "aws_iam_role_policy" "verifier_state_machine" {
  role = aws_iam_role.verifier_state_machine.id

  name   = "InvokeLambda"
  policy = data.aws_iam_policy_document.verifier_state_machine.json
}

resource "aws_cloudwatch_log_group" "verifier_state_machine" {
  name              = "/aws/vendedlogs/states/tailfed/verifier"
  retention_in_days = 7
}

resource "aws_sfn_state_machine" "verifier" {
  name = "TailfedVerifier"
  type = "STANDARD"

  role_arn = aws_iam_role.verifier_state_machine.arn

  logging_configuration {
    log_destination        = "${aws_cloudwatch_log_group.verifier_state_machine.arn}:*"
    level                  = local.state_machine_log_level_conversion[var.log_level]
    include_execution_data = true
  }

  definition = jsonencode({
    Comment       = "Periodically attempt to verify a host"
    StartAt       = "Initialize"
    QueryLanguage = "JSONata"

    States = {
      Initialize = {
        Type = "Pass"
        Next = "ForEachAddress"
        Assign = {
          retries    = 0
          maxRetries = 5
          waitTime   = 1
          waitJitter = 0.3
        }
      }

      ForEachAddress = {
        Type  = "Map"
        Next  = "CheckResult"
        Items = "{% $states.context.Execution.Input.addresses %}"
        ItemSelector = {
          id      = "{% $states.context.Execution.Input.id %}"
          address = "{% $states.context.Map.Item.Value %}"
        }
        ItemProcessor = {
          ProcessorConfig = { Mode = "INLINE" }
          StartAt         = "AttemptVerification"
          States = {
            AttemptVerification = {
              Type     = "Task"
              End      = true
              Resource = "arn:aws:states:::lambda:invoke"
              Arguments = {
                FunctionName = "${module.verifier.arn}:$LATEST"
                Payload      = "{% $states.input %}"
              }
              Output = "{% $states.result.Payload %}"
              Retry = [
                {
                  Comment         = "On Lambda Failures"
                  IntervalSeconds = 1
                  MaxAttempts     = 3
                  BackoffRate     = 2
                  JitterStrategy  = "FULL"
                  ErrorEquals = [
                    "Lambda.ServiceException",
                    "Lambda.AWSLambdaException",
                    "Lambda.SdkClientException",
                    "Lambda.TooManyRequestsException",
                  ]
                }
              ]
            }
          }
        }
        Output = {
          success = "{% $reduce($states.result, function($acc, $v) { $acc or $v.success }, false) %}"
        }
      }

      CheckResult = {
        Type    = "Choice"
        Default = "Wait"
        Choices = [
          {
            Next      = "Success"
            Comment   = "Verification Successful?"
            Condition = "{% $states.input.success %}"
          },
          {
            Next      = "Fail"
            Comment   = "Max Retries Reached?"
            Condition = "{% $retries >= $maxRetries %}"
          },
        ]
      }

      Wait = {
        Type    = "Wait"
        Seconds = "{% $max([$ceil($waitTime), 1]) %}"
        Next    = "ForEachAddress"
        Assign = {
          "retries"  = "{% $retries + 1 %}",
          "waitTime" = "{% $waitTime * 2 * (1 + ($waitJitter * ($random() * 2 - 1))) %}"
        }
      }

      Success = { Type = "Succeed" }
      Fail    = { Type = "Fail" }
    },
  })
}

data "aws_iam_policy_document" "verifier_state_machine_trust_policy" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["states.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "verifier" {
  statement {
    sid    = "Storage"
    effect = "Allow"
    actions = [
      "dynamodb:GetItem",
      "dynamodb:PutItem",
    ]
    resources = [aws_dynamodb_table.storage.arn]
  }
}

data "aws_iam_policy_document" "verifier_state_machine" {
  statement {
    sid       = "InvokeLambda"
    effect    = "Allow"
    actions   = ["lambda:InvokeFunction"]
    resources = ["${module.verifier.arn}:$LATEST"]
  }

  statement {
    sid    = "LoggingEvents"
    effect = "Allow"
    actions = [
      "logs:DescribeLogGroups",
      "logs:PutLogEvents",
    ]
    resources = ["${aws_cloudwatch_log_group.verifier_state_machine.arn}:*"]
  }

  statement {
    sid    = "LoggingVendored"
    effect = "Allow"
    actions = [
      "logs:CreateLogDelivery",
      "logs:DeleteLogDelivery",
      "logs:GetLogDelivery",
      "logs:ListLogDeliveries",
      "logs:UpdateLogDelivery",
    ]
    resources = ["*"]
  }

  statement {
    sid    = "LoggingPolicies"
    effect = "Allow"
    actions = [
      "logs:DescribeLogGroups",
      "logs:DescribeResourcePolicies",
      "logs:PutResourcePolicy",
    ]
    resources = ["arn:aws:logs:${var.region}:${data.aws_caller_identity.current.account_id}:log-group:*"]
  }
}
