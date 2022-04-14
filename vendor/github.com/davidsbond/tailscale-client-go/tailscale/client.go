// Package tailscale contains a basic implementation of a client for the Tailscale HTTP api. Documentation is here:
// https://github.com/tailscale/tailscale/blob/main/api.md
package tailscale

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/tailscale/hujson"
)

type (
	// The Client type is used to perform actions against the Tailscale API.
	Client struct {
		apiKey  string
		http    *http.Client
		baseURL *url.URL
		tailnet string
	}

	// The APIError type describes an error as returned by the Tailscale API.
	APIError struct {
		Message string         `json:"message"`
		Data    []APIErrorData `json:"data"`
		status  int
	}

	// The APIErrorData type describes elements of the data field within errors returned by the Tailscale API.
	APIErrorData struct {
		User   string   `json:"user"`
		Errors []string `json:"errors"`
	}

	// The ClientOption type is a function that is used to modify a Client.
	ClientOption func(c *Client) error
)

const baseURL = "https://api.tailscale.com"
const contentType = "application/json"

// NewClient returns a new instance of the Client type that will perform operations against a chosen tailnet and will
// provide the apiKey for authorization. Additional options can be provided, see ClientOption for more details.
func NewClient(apiKey, tailnet string, options ...ClientOption) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	c := &Client{
		apiKey:  apiKey,
		http:    &http.Client{Timeout: time.Minute},
		baseURL: u,
		tailnet: tailnet,
	}

	for _, option := range options {
		if err = option(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// WithBaseURL sets a custom baseURL for the Tailscale API, this is primarily used for testing purposes.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		u, err := url.Parse(baseURL)
		if err != nil {
			return err
		}

		c.baseURL = u
		return nil
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
	DERPMap   *ACLDERPMap         `json:"derpMap,omitempty" hujson:"DerpMap,omitempty"`
	Tests     []ACLTest           `json:"tests,omitempty" hujson:"Tests,omitempty"`
}

type ACLEntry struct {
	Action      string   `json:"action" hujson:"Action"`
	Ports       []string `json:"ports" hujson:"Ports"`
	Users       []string `json:"users" hujson:"Users"`
	Source      []string `json:"src" hujson:"Src"`
	Destination []string `json:"dst" hujson:"Dst"`
	Protocol    string   `json:"proto" hujson:"Proto"`
}

type ACLTest struct {
	User        string   `json:"user" hujson:"User"`
	Allow       []string `json:"allow" hujson:"Allow"`
	Deny        []string `json:"deny" hujson:"Deny"`
	Source      string   `json:"src" hujson:"Src"`
	Accept      []string `json:"accept" hujson:"Accept"`
}

type ACLDERPMap struct {
	Regions            map[int]*ACLDERPRegion `json:"regions" hujson:"Regions"`
	OmitDefaultRegions bool                   `json:"omitDefaultRegions,omitempty" hujson:"OmitDefaultRegions,omitempty"`
}

type ACLDERPRegion struct {
	RegionID   int            `json:"regionID" hujson:"RegionID"`
	RegionCode string         `json:"regionCode" hujson:"RegionCode"`
	RegionName string         `json:"regionName" hujson:"RegionName"`
	Avoid      bool           `json:"avoid,omitempty" hujson:"Avoid,omitempty"`
	Nodes      []*ACLDERPNode `json:"nodes" hujson:"Nodes"`
}

type ACLDERPNode struct {
	Name             string `json:"name" hujson:"Name"`
	RegionID         int    `json:"regionID" hujson:"RegionID"`
	HostName         string `json:"hostName" hujson:"HostName"`
	CertName         string `json:"certName,omitempty" hujson:"CertName,omitempty"`
	IPv4             string `json:"ipv4,omitempty" hujson:"IPv4,omitempty"`
	IPv6             string `json:"ipv6,omitempty" hujson:"IPv6,omitempty"`
	STUNPort         int    `json:"stunPort,omitempty" hujson:"STUNPort,omitempty"`
	STUNOnly         bool   `json:"stunOnly,omitempty" hujson:"STUNOnly,omitempty"`
	DERPPort         int    `json:"derpPort,omitempty" hujson:"DERPPort,omitempty"`
	InsecureForTests bool   `json:"insecureForRests,omitempty" hujson:"InsecureForTests,omitempty"`
	STUNTestIP       string `json:"stunTestIP,omitempty" hujson:"STUNTestIP,omitempty"`
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

type Device struct {
	Addresses                 []string  `json:"addresses"`
	Name                      string    `json:"name"`
	ID                        string    `json:"id"`
	Authorized                bool      `json:"authorized"`
	User                      string    `json:"user"`
	Tags                      []string  `json:"tags"`
	KeyExpiryDisabled         bool      `json:"keyExpiryDisabled"`
	BlocksIncomingConnections bool      `json:"blocksIncomingConnections"`
	ClientVersion             string    `json:"clientVersion"`
	Created                   time.Time `json:"created"`
	Expires                   time.Time `json:"expires"`
	Hostname                  string    `json:"hostname"`
	IsExternal                bool      `json:"isExternal"`
	LastSeen                  time.Time `json:"lastSeen"`
	MachineKey                string    `json:"machineKey"`
	NodeKey                   string    `json:"nodeKey"`
	OS                        string    `json:"os"`
	UpdateAvailable           bool      `json:"updateAvailable"`
}

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

// AuthorizeDevice marks the specified device identifier as authorized to join the tailnet.
func (c *Client) AuthorizeDevice(ctx context.Context, deviceID string) error {
	const uriFmt = "/api/v2/device/%s/authorized"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), map[string]bool{
		"authorized": true,
	})
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// DeleteDevice deletes the device given its deviceID.
func (c *Client) DeleteDevice(ctx context.Context, deviceID string) error {
	const uriFmt = "/api/v2/device/%s"
	req, err := c.buildRequest(ctx, http.MethodDelete, fmt.Sprintf(uriFmt, deviceID), nil)
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

type (
	// The KeyCapabilities type describes the capabilities of an authentication key.
	KeyCapabilities struct {
		Devices struct {
			Create struct {
				Reusable  bool     `json:"reusable"`
				Ephemeral bool     `json:"ephemeral"`
				Tags      []string `json:"tags"`
			} `json:"create"`
		} `json:"devices"`
	}

	// The Key type describes an authentication key within the tailnet.
	Key struct {
		ID           string          `json:"id"`
		Key          string          `json:"key"`
		Created      time.Time       `json:"created"`
		Expires      time.Time       `json:"expires"`
		Capabilities KeyCapabilities `json:"capabilities"`
	}
)

// CreateKey creates a new authentication key with the capabilities selected via the KeyCapabilities type. Returns
// the generated key if successful.
func (c *Client) CreateKey(ctx context.Context, capabilities KeyCapabilities) (Key, error) {
	const uriFmt = "/api/v2/tailnet/%s/keys"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, c.tailnet), map[string]KeyCapabilities{
		"capabilities": capabilities,
	})
	if err != nil {
		return Key{}, err
	}

	var key Key
	return key, c.performRequest(req, &key)
}

// GetKey returns all information on a key whose identifier matches the one provided. This will not return the
// authentication key itself, just the metadata.
func (c *Client) GetKey(ctx context.Context, id string) (Key, error) {
	const uriFmt = "/api/v2/tailnet/%s/keys/%s"

	req, err := c.buildRequest(ctx, http.MethodGet, fmt.Sprintf(uriFmt, c.tailnet, id), nil)
	if err != nil {
		return Key{}, err
	}

	var key Key
	return key, c.performRequest(req, &key)
}

// DeleteKey removes an authentication key from the tailnet.
func (c *Client) DeleteKey(ctx context.Context, id string) error {
	const uriFmt = "/api/v2/tailnet/%s/keys/%s"

	req, err := c.buildRequest(ctx, http.MethodDelete, fmt.Sprintf(uriFmt, c.tailnet, id), nil)
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// SetDeviceTags updates the tags of a target device.
func (c *Client) SetDeviceTags(ctx context.Context, deviceID string, tags []string) error {
	const uriFmt = "/api/v2/device/%s/tags"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), map[string][]string{
		"tags": tags,
	})
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

type (
	// The DeviceKey type represents the properties of the key of an individual device within
	// the tailnet.
	DeviceKey struct {
		KeyExpiryDisabled bool `json:"keyExpiryDisabled"` // Whether or not this device's key will ever expire.
		Preauthorized     bool `json:"preauthorized"`     // Whether or not this device is pre-authorized for the tailnet.
	}
)

// SetDeviceKey updates the properties of a device's key.
func (c *Client) SetDeviceKey(ctx context.Context, deviceID string, key DeviceKey) error {
	const uriFmt = "/api/v2/device/%s/key"

	req, err := c.buildRequest(ctx, http.MethodPost, fmt.Sprintf(uriFmt, deviceID), key)
	if err != nil {
		return err
	}

	return c.performRequest(req, nil)
}

// IsNotFound returns true if the provided error implementation is an APIError with a status of 404.
func IsNotFound(err error) bool {
	var apiErr APIError
	if errors.As(err, &apiErr) {
		return apiErr.status == http.StatusNotFound
	}

	return false
}

// ErrorData returns the contents of the APIError.Data field from the provided error if it is of type APIError. Returns
// a nil slice if the given error is not of type APIError.
func ErrorData(err error) []APIErrorData {
	var apiErr APIError
	if errors.As(err, &apiErr) {
		return apiErr.Data
	}

	return nil
}
