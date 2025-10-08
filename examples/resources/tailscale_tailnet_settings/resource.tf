resource "tailscale_tailnet_settings" "sample_tailnet_settings" {
  acls_externally_managed_on                  = true
  acls_external_link                          = "https://github.com/octocat/Hello-World"
  devices_approval_on                         = true
  devices_auto_updates_on                     = true
  devices_key_duration_days                   = 5
  users_approval_on                           = true
  users_role_allowed_to_join_external_tailnet = "member"
  posture_identity_collection_on              = true
  https_enabled                               = true
}
