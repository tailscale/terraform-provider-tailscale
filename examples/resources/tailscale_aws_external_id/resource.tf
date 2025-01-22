resource "tailscale_aws_external_id" "prod" {}

resource "tailscale_logstream_configuration" "configuration_logs" {
  log_type               = "configuration"
  destination_type       = "s3"
  s3_bucket              = aws_s3_bucket.tailscale_logs.id
  s3_region              = "us-west-2"
  s3_authentication_type = "rolearn"
  s3_role_arn            = aws_iam_role.logs_writer.arn
  s3_external_id         = tailscale_aws_external_id.prod.external_id
}

resource "aws_iam_role" "logs_writer" {
  name               = "logs-writer"
  assume_role_policy = data.aws_iam_policy_document.tailscale_assume_role.json
}

resource "aws_iam_role_policy" "logs_writer" {
  role   = aws_iam_role.logs_writer.id
  policy = data.aws_iam_policy_document.logs_writer.json
}

data "aws_iam_policy_document" "tailscale_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type = "AWS"
      identifiers = [tailscale_aws_external_id.prod.tailscale_aws_account_id]
    }
    condition {
      test     = "StringEquals"
      variable = "sts:ExternalId"
      values   = [tailscale_aws_external_id.prod.external_id]
    }
  }
}

data "aws_iam_policy_document" "logs_writer" {
  statement {
    effect = "Allow"
    actions = ["s3:*"]
    resources = [
      "arn:aws:s3:::example-bucket",
      "arn:aws:s3:::example-bucket/*"
    ]
  }
}
