// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"tailscale.com/client/tailscale/v2"
)

var (
	_ resource.Resource                = &contactsResource{}
	_ resource.ResourceWithImportState = &contactsResource{}
)

const resourceContactsDescription = `The contacts resource allows you to configure contact details for your Tailscale network. See https://tailscale.com/kb/1224/contact-preferences for more information.

Destroying this resource does not unset or modify values in the tailscale control plane, and simply removes the resource from Terraform state.
`

// NewContactsResource returns a new webhook resource.
func NewContactsResource() resource.Resource {
	return &contactsResource{}
}

type contactsResource struct {
	ResourceBase
}

// Metadata defines the resource name as it appears in Terraform configurations.
func (r *contactsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contacts"
}

// Schema defines a schema describing what fields can be defined in the resource.
func (r *contactsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: resourceContactsDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"account": schema.ListNestedBlock{
				Description: "Configuration for communications about important changes to your tailnet",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"email": schema.StringAttribute{
							Description: "Email address to send communications to",
							Required:    true,
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.SizeAtMost(1),
				},
			},
			"support": schema.ListNestedBlock{
				Description: "Configuration for communications about misconfigurations in your tailnet",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"email": schema.StringAttribute{
							Description: "Email address to send communications to",
							Required:    true,
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.SizeAtMost(1),
				},
			},
			"security": schema.ListNestedBlock{
				Description: "Configuration for communications about security issues affecting your tailnet",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"email": schema.StringAttribute{
							Description: "Email address to send communications to",
							Required:    true,
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.SizeAtMost(1),
				},
			},
		},
	}
}

type contactsResourceData struct {
	ID              types.String   `tfsdk:"id"`
	ContactAccount  []contactModel `tfsdk:"account"`
	ContactSupport  []contactModel `tfsdk:"support"`
	ContactSecurity []contactModel `tfsdk:"security"`
}

type contactModel struct {
	Email types.String `tfsdk:"email"`
}

func (r *contactsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan contactsResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, contactType := range []tailscale.ContactType{
		tailscale.ContactAccount,
		tailscale.ContactSupport,
		tailscale.ContactSecurity,
	} {
		r.updateContact(ctx, &plan, contactType, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	plan.ID = types.StringValue(createUUID())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *contactsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state contactsResourceData
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	contacts, err := r.Client.Contacts().Get(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error fetching contacts", err.Error())
		return
	}

	state.ContactAccount = []contactModel{{Email: types.StringValue(contacts.Account.Email)}}
	state.ContactSupport = []contactModel{{Email: types.StringValue(contacts.Support.Email)}}
	state.ContactSecurity = []contactModel{{Email: types.StringValue(contacts.Security.Email)}}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *contactsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan contactsResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, contactType := range []tailscale.ContactType{
		tailscale.ContactAccount,
		tailscale.ContactSupport,
		tailscale.ContactSecurity,
	} {
		r.updateContact(ctx, &plan, contactType, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	diags := resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *contactsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Deleting is a no-op since we cannot have unset contact information.
	// Deletion in this context is simply removing from terraform state.
	const diagDetail = `This resource has been successfully destroyed, but values in tailscale will remain set.
See https://tailscale.com/kb/1224/contact-preferences to learn more.`

	resp.Diagnostics.AddWarning(
		"Destroying tailscale_contacts does not unset contact values on tailscale",
		diagDetail,
	)
}

func (r *contactsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// updateContact calls the Tailscale API to update the contact configuration.
func (r *contactsResource) updateContact(ctx context.Context, data *contactsResourceData, contactType tailscale.ContactType, diags *diag.Diagnostics) {
	var contactEmail string
	switch contactType {
	case tailscale.ContactAccount:
		contactEmail = data.ContactAccount[0].Email.ValueString()
	case tailscale.ContactSupport:
		contactEmail = data.ContactSupport[0].Email.ValueString()
	case tailscale.ContactSecurity:
		contactEmail = data.ContactSecurity[0].Email.ValueString()
	}

	if err := r.Client.Contacts().Update(ctx, contactType, tailscale.UpdateContactRequest{Email: &contactEmail}); err != nil {
		diags.AddError("Failed to update contacts", err.Error())
		return
	}
}
