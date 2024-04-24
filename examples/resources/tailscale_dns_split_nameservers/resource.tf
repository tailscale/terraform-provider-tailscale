resource "tailscale_dns_split_nameservers" "sample_split_nameservers" {
  domain = "foo.example.com"

  nameservers = ["1.1.1.1"]
}
