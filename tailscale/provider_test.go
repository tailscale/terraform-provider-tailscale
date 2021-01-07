package tailscale_test

import (
	"os"
	"testing"

	"github.com/davidsbond/terraform-provider-tailscale/tailscale"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = tailscale.Provider()
	testAccProviders = map[string]*schema.Provider{
		"tailscale": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := tailscale.Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ *schema.Provider = tailscale.Provider()
}

func testAccPreCheck(t *testing.T) {
	if err := os.Getenv("TAILSCALE_API_KEY"); err == "" {
		t.Fatal("TAILSCALE_API_KEY must be set for acceptance tests")
	}
	if err := os.Getenv("TAILSCALE_DOMAIN"); err == "" {
		t.Fatal("TAILSCALE_DOMAIN must be set for acceptance tests")
	}
}
