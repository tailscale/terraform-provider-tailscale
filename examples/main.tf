terraform {
  required_providers {
    tailscale = {
      version = "0.1"
      source = "github.com/davidsbond/tailscale"
    }
  }
}

provider "tailscale" {
  api_key = "my-api-key"
  domain = "my-domain"
}

resource "tailscale_acl" "sample_acl" {
  acl = <<EOF
  {
    "acls": [
        {
            "action": "accept",
            "users": ["*"],
            "ports": ["*:*"]
        }
    ]
  }
EOF
}


resource "tailscale_dns_nameservers" "sample_nameservers" {
  nameservers = [
    "8.8.8.8",
    "8.8.4.4",
  ]
}

resource "tailscale_dns_preferences" "sample_preferences" {
  depends_on = [
    tailscale_dns_nameservers.sample_nameservers,
  ]

  magic_dns = true
}

resource "tailscale_dns_search_paths" "sample_search_paths" {
  search_paths = [
    "example.com",
  ]
}

output "sample_acl" {
  value = tailscale_acl.sample_acl.acl
}

output "sample_nameservers" {
  value = tailscale_dns_nameservers.sample_nameservers.nameservers
}

output "sample_preferences_magic_dns" {
  value = tailscale_dns_preferences.sample_preferences.magic_dns
}
