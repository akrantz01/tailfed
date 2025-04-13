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

  name = "verifier"
  arch = var.architecture

  bucket   = module.artifacts_proxy.id
  checksum = aws_s3_object_copy.artifacts["verifier"].checksum_sha256

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
        Next = "AttemptVerification"
        Assign = {
          index      = 0
          retries    = 0
          maxRetries = 8
          waitTime   = 1
          waitJitter = 0.3
        }
      }

      AttemptVerification = {
        Type     = "Task"
        Next     = "CheckResult"
        Resource = "arn:aws:states:::lambda:invoke"
        Arguments = {
          FunctionName = "${module.verifier.arn}:$LATEST"
          Payload = {
            id      = "{% $states.context.Execution.Input.id %}"
            address = "{% $states.context.Execution.Input.addresses[$index] %}"
          }
        }
        Output = "{% $states.result.Payload %}"
        Assign = {
          index = "{% ($index + 1) % $count($states.input.addresses) %}"
        }
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
        Seconds = "{% $max($waitTime, 1) %}"
        Next    = "AttemptVerification"
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
