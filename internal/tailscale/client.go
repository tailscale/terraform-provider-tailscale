// Package tailscale contains a basic implementation of a client for the Tailscale HTTP api. Documentation is here:
// https://github.com/tailscale/tailscale/blob/main/api.md
package tailscale

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type (
	Client struct {
		apiKey  string
		http    *http.Client
		baseURL *url.URL
		domain  string
	}

	APIError struct {
		Message string `json:"message"`
		status  int
	}
)

const baseURL = "https://api.tailscale.com"
const contentType = "application/json"

func NewClient(apiKey, domain string) *Client {
	u, err := url.Parse(baseURL)
	if err != nil {
		panic(err)
	}

	return &Client{
		apiKey:  apiKey,
		http:    &http.Client{Timeout: time.Minute},
		baseURL: u,
		domain:  domain,
	}
}

func (c *Client) buildRequest(ctx context.Context, method, uri string, body interface{}) (*http.Request, error) {
	u, err := c.baseURL.Parse(uri)
	if err != nil {
		return nil, err
	}

	var bodyBytes []byte
	if body != nil {
		bodyBytes, err = json.MarshalIndent(body, "", " ")
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
		if err = json.NewDecoder(res.Body).Decode(&apiErr); err != nil {
			return err
		}

		apiErr.status = res.StatusCode
		return apiErr
	}

	if out != nil {
		return json.NewDecoder(res.Body).Decode(out)
	}

	return nil
}

func (err APIError) Error() string {
	return fmt.Sprintf("%s (%v)", err.Message, err.status)
}

type DomainSearchPaths struct {
	SearchPaths []string `json:"searchPaths"`
}

// SetDNSSearchPaths replaces the list of search paths with the list supplied by the user and returns an error otherwise.
func (c *Client) SetDNSSearchPaths(ctx context.Context, searchPaths []string) error {
	const uriFmt = "/api/v2/domain/%v/dns/searchpaths"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.domain), DomainSearchPaths{
		SearchPaths: searchPaths,
	})
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// DNSSearchPaths retrieves the list of search paths that is currently set for the given domain.
func (c *Client) DNSSearchPaths(ctx context.Context) ([]string, error) {
	const uriFmt = "/api/v2/domain/%v/dns/searchpaths"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.domain), nil)
	if err != nil {
		return nil, err
	}

	var resp DomainSearchPaths
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp.SearchPaths, nil
}

type DomainDNSNameservers struct {
	DNS []string `json:"dns"`
}

// SetDNSNameservers replaces the list of DNS nameservers for the given domain with the list supplied by the user. Note
// that changing the list of DNS nameservers may also affect the status of MagicDNS (if MagicDNS is on).
func (c *Client) SetDNSNameservers(ctx context.Context, dns []string) error {
	const uriFmt = "/api/v2/domain/%v/dns/nameservers"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.domain), DomainDNSNameservers{
		DNS: dns,
	})
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// DNSNameservers lists the DNS nameservers for a domain
func (c *Client) DNSNameservers(ctx context.Context) ([]string, error) {
	const uriFmt = "/api/v2/domain/%v/dns/nameservers"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.domain), nil)
	if err != nil {
		return nil, err
	}

	var resp DomainDNSNameservers
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return resp.DNS, nil
}

type DomainACL struct {
	ACLs   []DomainACLEntry    `json:"acls"`
	Groups map[string][]string `json:"groups,omitempty"`
	Hosts  map[string]string   `json:"hosts,omitempty"`
}

type DomainACLEntry struct {
	Action string   `json:"action"`
	Ports  []string `json:"ports"`
	Users  []string `json:"users"`
}

// ACL retrieves the ACL that is currently set for the given domain.
func (c *Client) ACL(ctx context.Context) (*DomainACL, error) {
	const uriFmt = "/api/v2/domain/%s/acl"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.domain), nil)
	if err != nil {
		return nil, err
	}

	var resp DomainACL
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetACL sets the ACL for the given domain.
func (c *Client) SetACL(ctx context.Context, acl DomainACL) error {
	const uriFmt = "/api/v2/domain/%s/acl"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.domain), acl)
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

type DNSPreferences struct {
	MagicDNS bool `json:"magicDNS"`
}

// DNSPreferences retrieves the DNS preferences that are currently set for the given domain. Supply the domain of
// interest in the path.
func (c *Client) DNSPreferences(ctx context.Context) (*DNSPreferences, error) {
	const uriFmt = "/api/v2/domain/%s/dns/preferences"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.domain), nil)
	if err != nil {
		return nil, err
	}

	var resp DNSPreferences
	if err = c.performRequest(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetDNSPreferences replaces the DNS preferences for a domain, specifically, the MagicDNS setting. Note that MagicDNS
// is dependent on DNS servers.
func (c *Client) SetDNSPreferences(ctx context.Context, preferences DNSPreferences) error {
	const uriFmt = "/api/v2/domain/%s/dns/preferences"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.domain), preferences)
	if err != nil {
		return nil
	}

	return c.performRequest(req, nil)
}
