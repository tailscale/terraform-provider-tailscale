resource "tailscale_dns_configuration" "sample_configuration" {
    nameservers {
        address            = "8.8.8.8"
    }
    nameservers {
        address            = "1.1.1.1"
        use_with_exit_node = true
    }
    split_dns {
        domain             = "foo.example.com"
        nameservers {
            address            = "1.1.1.2"
            use_with_exit_node = true
        }
        nameservers {
            address            = "1.1.1.3"
        }
    }
    split_dns {
        domain             = "bar.example.com"
        nameservers {
            address            = "8.8.8.2"
            use_with_exit_node = true
        }
    }
    search_paths       = ["example.com", "anotherexample.com"]
    override_local_dns = true
    magic_dns = true
}