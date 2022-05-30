resource "tailscale_dns_search_paths" "sample_search_paths" {
  search_paths = [
    "example.com"
  ]
}
