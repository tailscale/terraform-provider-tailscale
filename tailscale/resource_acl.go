// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type aclResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	ACL                      types.String `tfsdk:"acl"`
	OverwriteExistingContent types.Bool   `tfsdk:"overwrite_existing_content"`
	ResetACLOnDestroy        types.Bool   `tfsdk:"reset_acl_on_destroy"`
}

// NewACLResource returns a new ACL resource.
func NewACLResource() resource.Resource {
	return &aclResource{}
}

type aclResource struct {
	ResourceBase
}

// Metadata defines the resource name as it appears in Terraform configurations.
func (r *aclResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acl"
}

const resourceACLDescription = `The acl resource allows you to configure a Tailscale policy file. See https://tailscale.com/kb/1395/tailnet-policy-file for more information. Note that this resource will completely overwrite existing policy file contents for a given tailnet.

If tests are defined in the policy file (the top-level "tests" section), policy file validation will occur before creation and update operations are applied.`

// From https://github.com/hashicorp/terraform-plugin-sdk/blob/34d8a9ebca6bed68fddb983123d6fda72481752c/internal/configs/hcl2shim/values.go#L19
// TODO: use an exported variable when https://github.com/hashicorp/terraform-plugin-sdk/issues/803 has been addressed.
const UnknownVariableValue = "74D93920-ED26-11E3-AC10-0800200C9A66"

func (r *aclResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: resourceACLDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"acl": schema.StringAttribute{
				Required:    true,
				Description: "The policy that defines which devices and users are allowed to connect in your network. Can be either a JSON or a HuJSON string.",
				PlanModifiers: []planmodifier.String{
					aclHuJSONModifier{},
				},
				Validators: []validator.String{
					aclHuJSONValidator{},
				},
			},
			"overwrite_existing_content": schema.BoolAttribute{
				Computed:    true,
				Optional:    true,
				Description: "If true, will skip requirement to import acl before allowing changes. Be careful, can cause the policy file to be overwritten",
				Default:     booldefault.StaticBool(false),
			},
			"reset_acl_on_destroy": schema.BoolAttribute{
				Computed:    true,
				Optional:    true,
				Description: "If true, will reset the policy file for the Tailnet to the default when this resource is destroyed",
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *aclResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state aclResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acl, err := r.Client.PolicyFile().Raw(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch ACL", err.Error())
		return
	}

	state.ACL = types.StringValue(acl.HuJSON)

	if state.ResetACLOnDestroy.IsNull() {
		state.ResetACLOnDestroy = types.BoolValue(false)
	}
	if state.OverwriteExistingContent.IsNull() {
		state.OverwriteExistingContent = types.BoolValue(false)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *aclResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan aclResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Setting the `ts-default` ETag will make this operation succeed only if
	// ACL contents has never been changed from its default value.
	var etag string
	if !plan.OverwriteExistingContent.ValueBool() {
		etag = "ts-default"
	}

	if err := r.Client.PolicyFile().Set(ctx, plan.ACL.ValueString(), etag); err != nil {
		if strings.HasSuffix(err.Error(), "(412)") {
			resp.Diagnostics.AddError("Overwrite Protected",
				"You are trying to overwrite a non-default policy. Please import the ACL first or set overwrite_existing_content = true.")
			return
		}
		resp.Diagnostics.AddError("Failed to set ACL", err.Error())
		return
	}

	plan.ID = types.StringValue(createUUID())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *aclResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan aclResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.Client.PolicyFile().Set(ctx, plan.ACL.ValueString(), "")
	if err != nil {
		resp.Diagnostics.AddError("Failed to update ACL", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *aclResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state aclResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	// Each tailnet always has an associated ACL file, so deleting a resource will
	// only remove it from Terraform state, leaving ACL contents intact.
	if !state.ResetACLOnDestroy.ValueBool() {
		return
	}

	// Setting the ACL to an empty string resets its value to the default.
	if err := r.Client.PolicyFile().Set(ctx, "", ""); err != nil {
		resp.Diagnostics.AddError("Failed to reset ACL", err.Error())
	}
}

func (r *aclResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
