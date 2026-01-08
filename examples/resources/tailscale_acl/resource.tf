resource "tailscale_acl" "as_json" {
  acl = jsonencode({
    grants = [
      {
        // Allow all users access to all ports.
        src = ["*"],
        dst = ["*"],
        ip  = ["*"],
      },
    ],
  })
}

resource "tailscale_acl" "as_hujson" {
  acl = <<EOF
  {
    // Comments in HuJSON policy are preserved when the policy is applied.
    "grants": [
      {
        // Allow all users access to all ports.
        "src": ["*"],
        "dst": ["*"],
        "ip": ["*"],
      },
    ],
  }
  EOF
}
