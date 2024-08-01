package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/tailscale/tailscale-client-go/tailscale"
)

func resourceWebhook() *schema.Resource {
	return &schema.Resource{
		Description:   "The webhook resource allows you to configure webhook endpoints for your Tailscale network. See https://tailscale.com/kb/1213/webhooks for more information.",
		ReadContext:   resourceWebhookRead,
		CreateContext: resourceWebhookCreate,
		UpdateContext: resourceWebhookUpdate,
		DeleteContext: resourceWebhookDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"endpoint_url": {
				Type:        schema.TypeString,
				Description: "The endpoint to send webhook events to.",
				Required:    true,
				ForceNew:    true,
			},
			"provider_type": {
				Type:        schema.TypeString,
				Description: "The provider type of the endpoint URL. Also referred to as the 'destination' for the webhook in the admin panel. Webhook event payloads are formatted according to the provider type if it is set to a known value. Must be one of `slack`, `mattermost`, `googlechat`, or `discord` if set.",
				Optional:    true,
				ForceNew:    true,
				ValidateFunc: validation.StringInSlice(
					[]string{
						string(tailscale.WebhookEmptyProviderType),
						string(tailscale.WebhookSlackProviderType),
						string(tailscale.WebhookMattermostProviderType),
						string(tailscale.WebhookGoogleChatProviderType),
						string(tailscale.WebhookDiscordProviderType),
					},
					false,
				),
			},
			"subscriptions": {
				Type:        schema.TypeSet,
				Description: "The Tailscale events to subscribe this webhook to. See https://tailscale.com/kb/1213/webhooks#events for the list of valid events.",
				Required:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice(
						[]string{
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
							string(tailscale.WebhookSubnetIPForwardingNotEnabled),
							string(tailscale.WebhookExitNodeIPForwardingNotEnabled),
						},
						false,
					),
				},
			},
			"secret": {
				Type:        schema.TypeString,
				Description: "The secret used for signing webhook payloads. Only set on resource creation. See https://tailscale.com/kb/1213/webhooks#webhook-secret for more information.",
				Sensitive:   true,
				Computed:    true,
			},
		},
	}
}

func resourceWebhookCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	endpointURL := d.Get("endpoint_url").(string)
	providerType := tailscale.WebhookProviderType(d.Get("provider_type").(string))
	subscriptions := d.Get("subscriptions").(*schema.Set).List()

	var requestSubscriptions []tailscale.WebhookSubscriptionType
	for _, subscription := range subscriptions {
		requestSubscriptions = append(requestSubscriptions, tailscale.WebhookSubscriptionType(subscription.(string)))
	}

	request := tailscale.CreateWebhookRequest{
		EndpointURL:   endpointURL,
		ProviderType:  providerType,
		Subscriptions: requestSubscriptions,
	}

	webhook, err := client.CreateWebhook(ctx, request)
	if err != nil {
		return diagnosticsError(err, "Failed to create webhook")
	}

	d.SetId(webhook.EndpointID)
	// Secret is only returned on create.
	d.Set("secret", webhook.Secret)

	return resourceWebhookRead(ctx, d, m)
}

func resourceWebhookRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	webhook, err := client.Webhook(ctx, d.Id())
	if err != nil {
		return diagnosticsError(err, "Failed to fetch webhook")
	}

	if err = d.Set("endpoint_url", webhook.EndpointURL); err != nil {
		return diagnosticsError(err, "Failed to set endpoint_url field")
	}

	if err = d.Set("provider_type", webhook.ProviderType); err != nil {
		return diagnosticsError(err, "Failed to set provider_type field")
	}

	if err = d.Set("subscriptions", webhook.Subscriptions); err != nil {
		return diagnosticsError(err, "Failed to set subscriptions field")
	}

	if err = d.Set("secret", d.Get("secret").(string)); err != nil {
		return diagnosticsError(err, "Failed to set secret field")
	}

	return nil
}

func resourceWebhookUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if !d.HasChange("subscriptions") {
		return resourceWebhookRead(ctx, d, m)
	}

	client := m.(*tailscale.Client)
	subscriptions := d.Get("subscriptions").(*schema.Set).List()

	var requestSubscriptions []tailscale.WebhookSubscriptionType
	for _, subscription := range subscriptions {
		requestSubscriptions = append(requestSubscriptions, tailscale.WebhookSubscriptionType(subscription.(string)))
	}

	_, err := client.UpdateWebhook(ctx, d.Id(), requestSubscriptions)
	if err != nil {
		return diagnosticsError(err, "Failed to update webhook")
	}

	return resourceWebhookRead(ctx, d, m)
}

func resourceWebhookDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	err := client.DeleteWebhook(ctx, d.Id())
	if err != nil {
		return diagnosticsError(err, "Failed to delete webhook")
	}

	return nil
}
