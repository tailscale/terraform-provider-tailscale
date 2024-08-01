resource "tailscale_contacts" "sample_contacts" {
  account {
    email = "account@example.com"
  }

  support {
    email = "support@example.com"
  }

  security {
    email = "security@example.com"
  }
}
