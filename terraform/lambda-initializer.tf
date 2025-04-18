module "initializer" {
  source = "./modules/lambda"

  depends_on = [aws_s3_object_copy.artifacts["initializer"]]

  name = "initializer"
  arch = var.architecture

  bucket   = module.artifacts_proxy.id
  checksum = local.artifact_hashes["initializer"]

  environment = {
    TAILFED_LOG_LEVEL                      = var.log_level
    TAILFED_LAUNCHER__STATE_MACHINE        = aws_sfn_state_machine.verifier.arn
    TAILFED_STORAGE__TABLE                 = aws_dynamodb_table.storage.arn
    TAILFED_TAILSCALE__TAILNET             = var.tailscale.tailnet
    TAILFED_TAILSCALE__API_KEY             = var.tailscale.api_key
    TAILFED_TAILSCALE__OAUTH_CLIENT_ID     = var.tailscale.oauth.client_id
    TAILFED_TAILSCALE__OAUTH_CLIENT_SECRET = var.tailscale.oauth.client_secret
  }

  policies = merge({ Lambda = data.aws_iam_policy_document.initializer.json }, var.execution_role_policies)
}

module "initializer_apigateway" {
  source = "./modules/apigateway-lambda"

  rest_api = aws_api_gateway_rest_api.default
  resource = aws_api_gateway_resource.start

  function = module.initializer
}

data "aws_iam_policy_document" "initializer" {
  statement {
    sid    = "Launcher"
    effect = "Allow"
    actions = [
      "states:DescribeStateMachine",
      "states:StartExecution",
    ]
    resources = [aws_sfn_state_machine.verifier.arn]
  }

  statement {
    sid       = "Storage"
    effect    = "Allow"
    actions   = ["dynamodb:PutItem"]
    resources = [aws_dynamodb_table.storage.arn]
  }
}
