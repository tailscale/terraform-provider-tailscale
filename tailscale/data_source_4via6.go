package tailscale

import (
	"context"
	"net/netip"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"tailscale.com/net/tsaddr"
)

func dataSource4Via6() *schema.Resource {
	return &schema.Resource{
		Description: "The 4via6 data source is calculates an IPv6 prefix for a given site ID and IPv4 CIDR. See Tailscale documentation for [4via6 subnets](https://tailscale.com/kb/1201/4via6-subnets/) for more details.",
		ReadContext: dataSource4Via6Read,
		Schema: map[string]*schema.Schema{
			"site": {
				Type:         schema.TypeInt,
				Required:     true,
				Description:  "Site ID (between 0 and 255)",
				ValidateFunc: validation.IntBetween(0, 255),
			},
			"cidr": {
				Type:         schema.TypeString,
				Description:  "The IPv4 CIDR to map",
				Required:     true,
				ValidateFunc: validation.IsCIDR,
			},
			"ipv6": {
				Type:        schema.TypeString,
				Description: "The 4via6 mapped address",
				Computed:    true,
			},
		},
	}
}

func dataSource4Via6Read(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	site := uint32(d.Get("site").(int))
	cidr, err := netip.ParsePrefix(d.Get("cidr").(string))
	if err != nil {
		return diagnosticsError(err, "Provided CIDR is invalid")
	}

	via, err := tsaddr.MapVia(site, cidr)
	if err != nil {
		return diagnosticsError(err, "Failed to map 4via6 address")
	}

	mapped := via.String()

	d.SetId(mapped)

	if err = d.Set("ipv6", mapped); err != nil {
		return diagnosticsError(err, "Failed to set ipv6")
	}

	return nil
}
