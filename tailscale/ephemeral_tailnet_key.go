// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"tailscale.com/client/tailscale/v2"
)

var (
	_ ephemeral.EphemeralResource              = &tailnetKeyEphemeralResource{}
	_ ephemeral.EphemeralResourceWithConfigure = &tailnetKeyEphemeralResource{}
	_ ephemeral.EphemeralResourceWithClose     = &tailnetKeyEphemeralResource{}
)

type tailnetKeyEphemeralModel struct {
	Reusable      types.Bool   `tfsdk:"reusable"`
	Ephemeral     types.Bool   `tfsdk:"ephemeral"`
	Tags          types.Set    `tfsdk:"tags"`
	Preauthorized types.Bool   `tfsdk:"preauthorized"`
	Key           types.String `tfsdk:"key"`
	Expiry        types.Int64  `tfsdk:"expiry"`
	CreatedAt     types.String `tfsdk:"created_at"`
	ExpiresAt     types.String `tfsdk:"expires_at"`
	Description   types.String `tfsdk:"description"`
}

func NewTailnetKeyEphemeralResource() ephemeral.EphemeralResource {
	return &tailnetKeyEphemeralResource{}
}

type tailnetKeyEphemeralResource struct {
	EphemeralResourceBase
}

func (t *tailnetKeyEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tailnet_key"
}

func (t *tailnetKeyEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The tailnet_key ephemeral resource creates a pre-authentication key for use during a Terraform operation. The key is deleted when Terraform closes the ephemeral resource. Requires Terraform 1.10 or later.",
		Attributes: map[string]schema.Attribute{
			"reusable": schema.BoolAttribute{
				Optional:    true,
				Description: "Indicates if the key is reusable or single-use. Defaults to `false`.",
			},
			"ephemeral": schema.BoolAttribute{
				Optional:    true,
				Description: "Indicates if devices authenticated by the key are ephemeral. This is the Tailscale key capability, separate from the Terraform ephemeral resource type. Defaults to `false`.",
			},
			"tags": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "List of tags to apply to machines authenticated by the key.",
			},
			"preauthorized": schema.BoolAttribute{
				Optional:    true,
				Description: "Determines whether machines authenticated by the key are authorized for the tailnet by default. Defaults to `false`.",
			},
			"key": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The authentication key. This value is ephemeral and is not stored in Terraform plan or state.",
			},
			"expiry": schema.Int64Attribute{
				Optional:    true,
				Description: "The expiry of the key in seconds. Defaults to `7776000` (90 days).",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The creation timestamp of the key in RFC3339 format.",
			},
			"expires_at": schema.StringAttribute{
				Computed:    true,
				Description: "The expiry timestamp of the key in RFC3339 format.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "A description of the key consisting of alphanumeric characters. Defaults to `\"\"`.",
				Validators: []validator.String{
					stringvalidator.LengthAtMost(50),
				},
			},
		},
	}
}

func (t *tailnetKeyEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var plan tailnetKeyEphemeralModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Reusable.IsUnknown() || plan.Ephemeral.IsUnknown() || plan.Preauthorized.IsUnknown() ||
		plan.Tags.IsUnknown() || plan.Expiry.IsUnknown() || plan.Description.IsUnknown() {
		resp.Diagnostics.AddError("Unknown tailnet key configuration", "All tailnet key configuration values must be known when the ephemeral resource is opened.")
		return
	}

	var tags []string
	if !plan.Tags.IsNull() {
		resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	expiry := int64(7776000)
	if !plan.Expiry.IsNull() {
		expiry = plan.Expiry.ValueInt64()
	}

	var createKeyRequest tailscale.CreateKeyRequest
	createKeyRequest.Capabilities.Devices.Create.Reusable = !plan.Reusable.IsNull() && plan.Reusable.ValueBool()
	createKeyRequest.Capabilities.Devices.Create.Ephemeral = !plan.Ephemeral.IsNull() && plan.Ephemeral.ValueBool()
	createKeyRequest.Capabilities.Devices.Create.Tags = tags
	createKeyRequest.Capabilities.Devices.Create.Preauthorized = !plan.Preauthorized.IsNull() && plan.Preauthorized.ValueBool()
	createKeyRequest.ExpirySeconds = expiry
	createKeyRequest.Description = plan.Description.ValueString()

	key, err := t.Client.Keys().CreateAuthKey(ctx, createKeyRequest)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create key", fmt.Sprintf("Error creating ephemeral tailnet key: %s", err))
		return
	}

	if key.ID == "" || key.Key == "" {
		if key.ID != "" {
			if deleteErr := t.Client.Keys().Delete(ctx, key.ID); deleteErr != nil && !tailscale.IsNotFound(deleteErr) {
				resp.Diagnostics.AddError("Failed to delete incomplete key", fmt.Sprintf("Error deleting incomplete ephemeral tailnet key with id %q: %s", key.ID, deleteErr))
			}
		}
		resp.Diagnostics.AddError("Incomplete key response", "The Tailscale API did not return both an ID and an authentication key.")
		return
	}

	plan.Key = types.StringValue(key.Key)
	plan.CreatedAt = types.StringValue(key.Created.Format(time.RFC3339))
	plan.ExpiresAt = types.StringValue(key.Expires.Format(time.RFC3339))

	privateKeyID, err := json.Marshal(key.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to encode key ID", fmt.Sprintf("Error encoding ephemeral tailnet key ID: %s", err))
		if deleteErr := t.Client.Keys().Delete(ctx, key.ID); deleteErr != nil && !tailscale.IsNotFound(deleteErr) {
			resp.Diagnostics.AddError("Failed to delete key after private state error", fmt.Sprintf("Error deleting ephemeral tailnet key with id %q: %s", key.ID, deleteErr))
		}
		return
	}

	resp.Diagnostics.Append(resp.Private.SetKey(ctx, "key_id", privateKeyID)...)
	if resp.Diagnostics.HasError() {
		if deleteErr := t.Client.Keys().Delete(ctx, key.ID); deleteErr != nil && !tailscale.IsNotFound(deleteErr) {
			resp.Diagnostics.AddError("Failed to delete key after private state error", fmt.Sprintf("Error deleting ephemeral tailnet key with id %q: %s", key.ID, deleteErr))
		}
		return
	}

	resp.Diagnostics.Append(resp.Result.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		if deleteErr := t.Client.Keys().Delete(ctx, key.ID); deleteErr != nil && !tailscale.IsNotFound(deleteErr) {
			resp.Diagnostics.AddError("Failed to delete key after result error", fmt.Sprintf("Error deleting ephemeral tailnet key with id %q: %s", key.ID, deleteErr))
		}
	}
}

func (t *tailnetKeyEphemeralResource) Close(ctx context.Context, req ephemeral.CloseRequest, resp *ephemeral.CloseResponse) {
	if req.Private == nil {
		resp.Diagnostics.AddError("Missing ephemeral resource private state", "Cannot delete the tailnet key because its private ID was not provided.")
		return
	}

	privateKeyID, diags := req.Private.GetKey(ctx, "key_id")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var keyID string
	if err := json.Unmarshal(privateKeyID, &keyID); err != nil {
		resp.Diagnostics.AddError("Invalid tailnet key ID", fmt.Sprintf("Unable to decode the private tailnet key ID: %s", err))
		return
	}
	if keyID == "" {
		resp.Diagnostics.AddError("Missing tailnet key ID", "Cannot delete the tailnet key because its private ID is empty.")
		return
	}

	err := t.Client.Keys().Delete(ctx, keyID)
	if err != nil && !tailscale.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete key", fmt.Sprintf("Error deleting ephemeral tailnet key with id %q: %s", keyID, err))
	}
}
