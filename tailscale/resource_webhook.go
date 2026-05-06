// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"tailscale.com/client/tailscale/v2"
)

var (
	_ resource.Resource                = &webhookResource{}
	_ resource.ResourceWithImportState = &webhookResource{}
)

// NewWebhookResource returns a new webhook resource.
func NewWebhookResource() resource.Resource {
	return &webhookResource{}
}

type webhookResource struct {
	ResourceBase
	ResourceImportedByID
}

// Metadata defines the resource name as it appears in Terraform configurations.
func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

// Schema defines a schema describing what fields can be defined in the resource.
func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The webhook resource allows you to configure webhook endpoints for your Tailscale network. See https://tailscale.com/kb/1213/webhooks for more information.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"endpoint_url": schema.StringAttribute{
				Description: "The endpoint to send webhook events to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"provider_type": schema.StringAttribute{
				Description: "The provider type of the endpoint URL. This determines the payload format sent to the destination. Valid values are `slack`, `mattermost`, `googlechat`, and `discord`.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(tailscale.WebhookEmptyProviderType),
						string(tailscale.WebhookSlackProviderType),
						string(tailscale.WebhookMattermostProviderType),
						string(tailscale.WebhookGoogleChatProviderType),
						string(tailscale.WebhookDiscordProviderType),
					),
				},
			},
			"subscriptions": schema.SetAttribute{
				Description: "The set of events that trigger this webhook. For a full list of event types, see the [webhooks documentation](https://tailscale.com/kb/1213/webhooks#events).",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.OneOf(
							string(tailscale.WebhookCategoryTailnetManagement),
							string(tailscale.WebhookNodeCreated),
							string(tailscale.WebhookNodeNeedsApproval),
							string(tailscale.WebhookNodeApproved),
							string(tailscale.WebhookNodeKeyExpiringInOneDay),
							string(tailscale.WebhookNodeKeyExpired),
							string(tailscale.WebhookNodeDeleted),
							string(tailscale.WebhookPolicyUpdate),
							string(tailscale.WebhookUserCreated),
							string(tailscale.WebhookUserNeedsApproval),
							string(tailscale.WebhookUserSuspended),
							string(tailscale.WebhookUserRestored),
							string(tailscale.WebhookUserDeleted),
							string(tailscale.WebhookUserApproved),
							string(tailscale.WebhookUserRoleUpdated),
							string(tailscale.WebhookCategoryDeviceMisconfigurations),
							string(tailscale.WebhookSubnetIPForwardingNotEnabled),
							string(tailscale.WebhookExitNodeIPForwardingNotEnabled),
						),
					),
				},
			},
			"secret": schema.StringAttribute{
				Description: "The secret used for signing webhook payloads. Only set on resource creation. See https://tailscale.com/kb/1213/webhooks#webhook-secret for more information.",
				Sensitive:   true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

type webhookResourceData struct {
	ID            types.String `tfsdk:"id"`
	Secret        types.String `tfsdk:"secret"`
	EndpointURL   types.String `tfsdk:"endpoint_url"`
	ProviderType  types.String `tfsdk:"provider_type"`
	Subscriptions types.Set    `tfsdk:"subscriptions"`
}

// requestSubscriptions gets a list of subscriptions in a type that
// can be passed to the Tailscale API.
func (d webhookResourceData) requestSubscriptions(ctx context.Context, diags diag.Diagnostics) []tailscale.WebhookSubscriptionType {
	var subscriptions []string
	diags.Append(d.Subscriptions.ElementsAs(ctx, &subscriptions, false)...)
	if diags.HasError() {
		return nil
	}
	var requestSubscriptions []tailscale.WebhookSubscriptionType
	for _, subscription := range subscriptions {
		requestSubscriptions = append(requestSubscriptions, tailscale.WebhookSubscriptionType(subscription))
	}
	return requestSubscriptions
}

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpointURL := plan.EndpointURL.ValueString()
	providerType := tailscale.WebhookProviderType(plan.ProviderType.ValueString())
	requestSubscriptions := plan.requestSubscriptions(ctx, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	request := tailscale.CreateWebhookRequest{
		EndpointURL:   endpointURL,
		ProviderType:  providerType,
		Subscriptions: requestSubscriptions,
	}

	webhook, err := r.Client.Webhooks().Create(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create webhook", err.Error())
		return
	}

	plan.ID = types.StringValue(webhook.EndpointID)

	// Secret is only returned on create.
	if secret := webhook.Secret; secret != nil {
		plan.Secret = types.StringValue(*secret)
	} else {
		resp.Diagnostics.AddError("Failed to get webhook secret", "Expected Create() call to return webhook secret, but got nil")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookResourceData
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhook, err := r.Client.Webhooks().Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error fetching webhook", err.Error())
		return
	}

	state.EndpointURL = types.StringValue(webhook.EndpointURL)
	state.ProviderType = StringValueNullIfEmpty(string(webhook.ProviderType))
	state.Subscriptions = SetOfStringValue(ctx, webhook.Subscriptions, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state webhookResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Subscriptions.Equal(state.Subscriptions) {
		return
	}

	endpointID := plan.ID.ValueString()
	requestSubscriptions := plan.requestSubscriptions(ctx, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.Client.Webhooks().Update(ctx, endpointID, requestSubscriptions)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update webhook", err.Error())
		return
	}

	diags := resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookResourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpointID := state.ID.ValueString()

	if err := r.Client.Webhooks().Delete(ctx, endpointID); err != nil {
		resp.Diagnostics.AddError("Failed to delete webhook", err.Error())
		return
	}
}
