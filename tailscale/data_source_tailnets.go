// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"tailscale.com/client/tailscale/v2"
)

var (
	_ datasource.DataSourceWithConfigure = &tailnetsDataSource{}
)

type tailnetsDataSource struct {
	DataSourceBase
}

type tailnetsDataSourceModel struct {
	ID       types.String             `tfsdk:"id"`
	Tailnets []tailnetDataSourceModel `tfsdk:"tailnets"`
}

type tailnetDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	DisplayName types.String `tfsdk:"display_name"`
	OrgID       types.String `tfsdk:"org_id"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func NewTailnetsDataSource() datasource.DataSource {
	return &tailnetsDataSource{}
}

func (d tailnetsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tailnets"
}

func (d tailnetsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The tailnets data source describes the tailnets in an organization using the alpha tailnet creation API. Provider OAuth credentials require the tailnets:read scope.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
		Blocks: map[string]schema.Block{
			"tailnets": schema.ListNestedBlock{
				Description: "The list of tailnets in the organization.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The stable, globally unique identifier for the tailnet.",
							Computed:    true,
						},
						"display_name": schema.StringAttribute{
							Description: "The display name for the tailnet.",
							Computed:    true,
						},
						"org_id": schema.StringAttribute{
							Description: "The stable, globally unique identifier for the organization.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "The creation timestamp of the tailnet in RFC3339 format.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d tailnetsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tailnetsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tailnets, err := listTailnets(ctx, d.Client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch tailnets", err.Error())
		return
	}

	data.Tailnets = make([]tailnetDataSourceModel, 0, len(tailnets))
	for _, tailnet := range tailnets {
		data.Tailnets = append(data.Tailnets, tailnetDataSourceModel{
			ID:          types.StringValue(tailnet.ID),
			DisplayName: types.StringValue(tailnet.DisplayName),
			OrgID:       types.StringValue(tailnet.OrgID),
			CreatedAt:   types.StringValue(tailnet.CreatedAt.Format(time.RFC3339)),
		})
	}
	data.ID = types.StringValue(createUUID())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// The upstream v2.11.0 List method cannot decode pagination metadata or follow cursors.
func listTailnets(ctx context.Context, client *tailscale.Client) ([]tailscale.Tailnet, error) {
	client.Tailnets() // Initialize the SDK's authenticated HTTP client and defaults.
	uri := client.BaseURL.JoinPath("/api/v2/organizations/-/tailnets")
	var tailnets []tailscale.Tailnet
	seen := make(map[string]bool)
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", client.UserAgent)
		if client.APIKey != "" {
			req.SetBasicAuth(client.APIKey, "")
		}
		resp, err := client.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			apiErr := tailscale.APIError{}
			err := json.NewDecoder(resp.Body).Decode(&apiErr)
			resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to list tailnets: HTTP %d: %w", resp.StatusCode, err)
			}
			apiErr.Status = resp.StatusCode
			return nil, apiErr
		}
		var page struct {
			Tailnets []tailscale.Tailnet `json:"tailnets"`
			Cursor   string              `json:"cursor"`
		}
		err = json.NewDecoder(resp.Body).Decode(&page)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if page.Tailnets == nil {
			return nil, fmt.Errorf("tailnet list response is missing the tailnets array")
		}
		tailnets = append(tailnets, page.Tailnets...)
		if page.Cursor == "" {
			return tailnets, nil
		}
		if seen[page.Cursor] {
			return nil, fmt.Errorf("tailnet list response repeated a pagination cursor")
		}
		seen[page.Cursor] = true
		query := uri.Query()
		query.Set("cursor", page.Cursor)
		uri.RawQuery = query.Encode()
	}
}
