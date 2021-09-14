// Package tailscale contains a basic implementation of a client for the Tailscale HTTP api. Documentation is here:
// https://github.com/tailscale/tailscale/blob/main/api.md
package tailscale

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/tailscale/hujson"
)

type (
	Client struct {
		apiKey  string
		http    *http.Client
		baseURL *url.URL
		tailnet string
	}

	APIError struct {
		Message string `json:"message"`
		status  int
	}
)

const baseURL = "https://api.tailscale.com"
const contentType = "application/json"

func NewClient(apiKey, tailnet string) *Client {
	u, err := url.Parse(baseURL)
	if err != nil {
		panic(err)
	}

	return &Client{
		apiKey:  apiKey,
		http:    &http.Client{Timeout: time.Minute},
		baseURL: u,
		tailnet: tailnet,
	}
}

func (c *Client) buildRequest(ctx context.Context, method, uri string, body interface{}) (*http.Request, error) {
	u, err := c.baseURL.Parse(uri)
	if err != nil {
		return nil, err
	}

	var bodyBytes []byte
	if body != nil {
		bodyBytes, err = hujson.MarshalIndent(body, "", " ")
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	switch {
	case body == nil:
		req.Header.Set("Accept", contentType)
	default:
		req.Header.Set("Content-Type", contentType)
	}

	req.SetBasicAuth(c.apiKey, "")
	return req, nil
}

func (c *Client) performRequest(req *http.Request, out interface{}) error {
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		var apiErr APIError
		if err = hujson.NewDecoder(res.Body).Decode(&apiErr); err != nil {
			return err
		}

		apiErr.status = res.StatusCode
		return apiErr
	}

	if out != nil {
		return hujson.NewDecoder(res.Body).Decode(out)
	}

	return nil
}

func (err APIError) Error() string {
	return fmt.Sprintf("%s (%v)", err.Message, err.status)
}

// SetDNSSearchPaths replaces the list of search paths with the list supplied by the user and returns an error otherwise.
func (c *Client) SetDNSSearchPaths(ctx context.Context, searchPaths []string) error {
	const uriFmt = "/api/v2/tailnet/%v/dns/searchpaths"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnet), map[string][]string{
		"searchPaths": searchPaths,
	})
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// DNSSearchPaths retrieves the list of search paths that is currently set for the given tailnet.
func (c *Client) DNSSearchPaths(ctx context.Context) ([]string, error) {
	const uriFmt = "/api/v2/tailnet/%v/dns/searchpaths"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet), nil)
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]string)
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp["searchPaths"], nil
}

// SetDNSNameservers replaces the list of DNS nameservers for the given tailnet with the list supplied by the user. Note
// that changing the list of DNS nameservers may also affect the status of MagicDNS (if MagicDNS is on).
func (c *Client) SetDNSNameservers(ctx context.Context, dns []string) error {
	const uriFmt = "/api/v2/tailnet/%v/dns/nameservers"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnet), map[string][]string{
		"dns": dns,
	})
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// DNSNameservers lists the DNS nameservers for a tailnet
func (c *Client) DNSNameservers(ctx context.Context) ([]string, error) {
	const uriFmt = "/api/v2/tailnet/%v/dns/nameservers"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet), nil)
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]string)
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp["dns"], nil
}

type ACL struct {
	ACLs      []ACLEntry          `json:"acls" hujson:"ACLs,omitempty"`
	Groups    map[string][]string `json:"groups,omitempty" hujson:"Groups,omitempty"`
	Hosts     map[string]string   `json:"hosts,omitempty" hujson:"Hosts,omitempty"`
	TagOwners map[string][]string `json:"tagowners,omitempty" hujson:"TagOwners,omitempty"`
	Tests     []ACLTest           `json:"tests,omitempty" hujson:"Tests,omitempty"`
}

type ACLEntry struct {
	Action string   `json:"action" hujson:"Action"`
	Ports  []string `json:"ports" hujson:"Ports"`
	Users  []string `json:"users" hujson:"Users"`
}

type ACLTest struct {
	User  string   `json:"user" hujson:"User"`
	Allow []string `json:"allow" hujson:"Allow"`
	Deny  []string `json:"deny" hujson:"Deny"`
}

// ACL retrieves the ACL that is currently set for the given tailnet.
func (c *Client) ACL(ctx context.Context) (*ACL, error) {
	const uriFmt = "/api/v2/tailnet/%s/acl"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet), nil)
	if err != nil {
		return nil, err
	}

	var resp ACL
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetACL sets the ACL for the given tailnet.
func (c *Client) SetACL(ctx context.Context, acl ACL) error {
	const uriFmt = "/api/v2/tailnet/%s/acl"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnet), acl)
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

type DNSPreferences struct {
	MagicDNS bool `json:"magicDNS"`
}

// DNSPreferences retrieves the DNS preferences that are currently set for the given tailnet. Supply the tailnet of
// interest in the path.
func (c *Client) DNSPreferences(ctx context.Context) (*DNSPreferences, error) {
	const uriFmt = "/api/v2/tailnet/%s/dns/preferences"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet), nil)
	if err != nil {
		return nil, err
	}

	var resp DNSPreferences
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetDNSPreferences replaces the DNS preferences for a tailnet, specifically, the MagicDNS setting. Note that MagicDNS
// is dependent on DNS servers.
func (c *Client) SetDNSPreferences(ctx context.Context, preferences DNSPreferences) error {
	const uriFmt = "/api/v2/tailnet/%s/dns/preferences"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnet), preferences)
	if err != nil {
		return nil
	}

	return c.performRequest(req, nil)
}

type (
	DeviceRoutes struct {
		Advertised []string `json:"advertisedRoutes"`
		Enabled    []string `json:"enabledRoutes"`
	}
)

// SetDeviceSubnetRoutes sets which subnet routes are enabled to be routed by a device by replacing the existing list
// of subnet routes with the supplied routes. Routes can be enabled without a device advertising them (e.g. for preauth).
func (c *Client) SetDeviceSubnetRoutes(ctx context.Context, deviceID string, routes []string) error {
	const uriFmt = "/api/v2/device/%s/routes"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), map[string][]string{
		"routes": routes,
	})
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// DeviceSubnetRoutes Retrieves the list of subnet routes that a device is advertising, as well as those that are
// enabled for it. Enabled routes are not necessarily advertised (e.g. for pre-enabling), and likewise, advertised
// routes are not necessarily enabled.
func (c *Client) DeviceSubnetRoutes(ctx context.Context, deviceID string) (*DeviceRoutes, error) {
	const uriFmt = "/api/v2/device/%s/routes"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, deviceID), nil)
	if err != nil {
		return nil, err
	}

	var resp DeviceRoutes
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

type (
	Device struct {
		Addresses []string `json:"addresses"`
		Name      string   `json:"name"`
		ID        string   `json:"id"`
	}
)

// Devices lists the devices in a tailnet.
func (c *Client) Devices(ctx context.Context) ([]Device, error) {
	const uriFmt = "/api/v2/tailnet/%s/devices"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet), nil)
	if err != nil {
		return nil, err
	}

	resp := make(map[string][]Device)
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp["devices"], nil
}
