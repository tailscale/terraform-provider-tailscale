resource "tailscale_dns_nameservers" "sample_nameservers" {
  nameservers = [
    "8.8.8.8",
    "8.8.4.4"
  ]
}
