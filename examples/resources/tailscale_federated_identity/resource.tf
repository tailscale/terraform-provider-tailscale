resource "tailscale_federated_identity" "example_federated_identity" {
  description = "Example federated identity"
  scopes      = ["auth_keys", "devices:core"]
  tags        = ["tag:test"]
  issuer      = "https://example.com"
  subject     = "example-sub-*"
  custom_claim_rules = {
    repo_name = "example-repo-name"
  }
}
