// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"tailscale.com/client/tailscale/v2"
)

var (
	_ resource.Resource                = &serviceResource{}
	_ resource.ResourceWithImportState = &webhookResource{}
)

type serviceResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Addrs   types.List   `tfsdk:"addrs"`
	Comment types.String `tfsdk:"comment"`
	Ports   types.Set    `tfsdk:"ports"`
	Tags    types.Set    `tfsdk:"tags"`
}

// NewServiceResource returns a new service resource.
func NewServiceResource() resource.Resource {
	return &serviceResource{}
}

type serviceResource struct {
	ResourceBase
}

// Metadata defines the resource name as it appears in Terraform configurations.
func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

// Schema defines a schema describing what fields can be defined in the resource.
func (r *serviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Service resource allows you to manage Tailscale Services in your Tailscale network. Services let you publish internal resources (like databases or web servers) as named resources in your tailnet. Services provide a stable MagicDNS name, a Tailscale virtual IP address pair, can be served by multiple nodes, and are valid access control destinations. See https://tailscale.com/docs/features/tailscale-services) for more information.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the Service. Must begin with `svc:`.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("^svc:.*"), "must start with svc:"),
				},
			},
			"id": schema.StringAttribute{
				// The ID must be predictable to support importing existing
				// Services, e.g. 'terraform import tailscale_service.my_service
				// svc:my-service'. The Service name will be a known value and
				// is the ID used by the API anyway.
				Description: "The Service name, e.g. 'svc:my-service'.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"addrs": schema.ListAttribute{
				Description: "The IP addresses assigned to the Service.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"comment": schema.StringAttribute{
				Description: "An optional comment describing the Service.",
				Optional:    true,
			},
			"ports": schema.SetAttribute{
				Description: "A list of protocol:port pairs to be exposed by the Service. The only supported protocol is \"tcp\" at this time. \"do-not-validate\" can be used to skip validation.",
				Required:    true,
				ElementType: types.StringType,
			},
			"tags": schema.SetAttribute{
				Description: "The ACL tags applied to the Service.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc := r.buildServiceFromResource(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.Client.VIPServices().CreateOrUpdate(ctx, svc); err != nil {
		resp.Diagnostics.AddError("Failed to create Service", err.Error())
		return
	}

	plan.ID = plan.Name

	// Re-fetch to get the 'addrs' which are computed by the API
	createdSvc, err := r.Client.VIPServices().Get(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch service for IPs", err.Error())
		return
	}

	addrs, d := types.ListValueFrom(ctx, types.StringType, createdSvc.Addrs)
	resp.Diagnostics.Append(d...)
	plan.Addrs = addrs

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.Client.VIPServices().Get(ctx, state.ID.ValueString())
	if err != nil {
		if tailscale.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to fetch Service", err.Error())
		return
	}

	state.Name = types.StringValue(svc.Name)
	state.Comment = types.StringValue(svc.Comment)

	addrs, d := types.ListValueFrom(ctx, types.StringType, svc.Addrs)
	resp.Diagnostics.Append(d...)
	state.Addrs = addrs

	ports, d := types.SetValueFrom(ctx, types.StringType, svc.Ports)
	resp.Diagnostics.Append(d...)
	state.Ports = ports

	tags, d := types.SetValueFrom(ctx, types.StringType, svc.Tags)
	resp.Diagnostics.Append(d...)
	state.Tags = tags

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc := r.buildServiceFromResource(ctx, &plan, &resp.Diagnostics)
	if err := r.Client.VIPServices().CreateOrUpdate(ctx, svc); err != nil {
		resp.Diagnostics.AddError("Failed to update Service", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	err := r.Client.VIPServices().Delete(ctx, state.ID.ValueString())
	if err != nil && !tailscale.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Service", err.Error())
	}
}

func (d serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *serviceResource) buildServiceFromResource(ctx context.Context, model *serviceResourceModel, diags *diag.Diagnostics) tailscale.VIPService {
	var ports, tags, addrs []string
	diags.Append(model.Ports.ElementsAs(ctx, &ports, false)...)
	diags.Append(model.Tags.ElementsAs(ctx, &tags, false)...)

	if !model.Addrs.IsNull() && !model.Addrs.IsUnknown() {
		diags.Append(model.Addrs.ElementsAs(ctx, &addrs, false)...)
	}

	return tailscale.VIPService{
		Name:    model.Name.ValueString(),
		Comment: model.Comment.ValueString(),
		Addrs:   addrs,
		Ports:   ports,
		Tags:    tags,
	}
}
