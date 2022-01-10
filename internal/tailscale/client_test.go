package tailscale_test

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/davidsbond/terraform-provider-tailscale/internal/tailscale"
	"github.com/stretchr/testify/assert"
	"github.com/tailscale/hujson"
)

var (
	//go:embed testdata/acl.json
	jsonACL []byte
	//go:embed testdata/acl.hujson
	huJSONACL []byte
)

func TestACL_Unmarshal(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name          string
		ACLContent    []byte
		Expected      tailscale.ACL
		UnmarshalFunc func(data []byte, v interface{}) error
	}{
		{
			Name:          "It should handle JSON ACLs",
			ACLContent:    jsonACL,
			UnmarshalFunc: json.Unmarshal,
			Expected: tailscale.ACL{
				ACLs: []tailscale.ACLEntry{
					{
						Action: "accept",
						Ports:  []string{"*:*"},
						Users:  []string{"*"},
					},
				},
				TagOwners: map[string][]string{
					"tag:example": {"group:example"},
				},
				Hosts: map[string]string{
					"example-host-1": "100.100.100.100",
					"example-host-2": "100.100.101.100/24",
				},
				DERPMap: &tailscale.ACLDERPMap{
					Regions: map[int]*tailscale.ACLDERPRegion{
						900: {
							RegionID:   900,
							RegionCode: "example",
							RegionName: "example",
							Nodes: []*tailscale.ACLDERPNode{
								{
									Name:     "1",
									RegionID: 900,
									HostName: "example.com",
								},
							},
						},
					},
					OmitDefaultRegions: false,
				},
				Groups: map[string][]string{
					"group:example": {
						"user1@example.com",
						"user2@example.com",
					},
				},
				Tests: []tailscale.ACLTest{
					{
						User:  "user1@example.com",
						Allow: []string{"example-host-1:22", "example-host-2:80"},
						Deny:  []string{"exapmle-host-2:100"},
					},
					{
						User:  "user2@example.com",
						Allow: []string{"100.60.3.4:22"},
					},
				},
			},
		},
		{
			Name:          "It should handle HuJSON ACLs",
			ACLContent:    huJSONACL,
			UnmarshalFunc: hujson.Unmarshal,
			Expected: tailscale.ACL{
				ACLs: []tailscale.ACLEntry{
					{
						Action: "accept",
						Ports:  []string{"*:*"},
						Users:  []string{"*"},
					},
				},
				TagOwners: map[string][]string{
					"tag:example": {"group:example"},
				},
				Hosts: map[string]string{
					"example-host-1": "100.100.100.100",
					"example-host-2": "100.100.101.100/24",
				},
				DERPMap: &tailscale.ACLDERPMap{
					Regions: map[int]*tailscale.ACLDERPRegion{
						900: {
							RegionID:   900,
							RegionCode: "example",
							RegionName: "example",
							Nodes: []*tailscale.ACLDERPNode{
								{
									Name:     "1",
									RegionID: 900,
									HostName: "example.com",
								},
							},
						},
					},
					OmitDefaultRegions: false,
				},
				Groups: map[string][]string{
					"group:example": {
						"user1@example.com",
						"user2@example.com",
					},
				},
				Tests: []tailscale.ACLTest{
					{
						User:  "user1@example.com",
						Allow: []string{"example-host-1:22", "example-host-2:80"},
						Deny:  []string{"exapmle-host-2:100"},
					},
					{
						User:  "user2@example.com",
						Allow: []string{"100.60.3.4:22"},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			var actual tailscale.ACL

			assert.NoError(t, tc.UnmarshalFunc(tc.ACLContent, &actual))
			assert.EqualValues(t, tc.Expected, actual)
		})
	}
}

func TestClient_SetACL(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK
	expectedACL := tailscale.ACL{
		ACLs: []tailscale.ACLEntry{
			{
				Action: "accept",
				Ports:  []string{"*:*"},
				Users:  []string{"*"},
			},
		},
		TagOwners: map[string][]string{
			"tag:example": {"group:example"},
		},
		Hosts: map[string]string{
			"example-host-1": "100.100.100.100",
			"example-host-2": "100.100.101.100/24",
		},
		Groups: map[string][]string{
			"group:example": {
				"user1@example.com",
				"user2@example.com",
			},
		},
		Tests: []tailscale.ACLTest{
			{
				User:  "user1@example.com",
				Allow: []string{"example-host-1:22", "example-host-2:80"},
				Deny:  []string{"exapmle-host-2:100"},
			},
			{
				User:  "user2@example.com",
				Allow: []string{"100.60.3.4:22"},
			},
		},
	}

	assert.NoError(t, client.SetACL(context.Background(), expectedACL))
	assert.Equal(t, http.MethodPost, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/acl", server.Path)

	var actualACL tailscale.ACL
	assert.NoError(t, json.Unmarshal(server.Body.Bytes(), &actualACL))
	assert.EqualValues(t, expectedACL, actualACL)
}

func TestClient_ACL(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)

	server.ResponseCode = http.StatusOK
	server.ResponseBody = &tailscale.ACL{
		ACLs: []tailscale.ACLEntry{
			{
				Action: "accept",
				Ports:  []string{"*:*"},
				Users:  []string{"*"},
			},
		},
		TagOwners: map[string][]string{
			"tag:example": {"group:example"},
		},
		Hosts: map[string]string{
			"example-host-1": "100.100.100.100",
			"example-host-2": "100.100.101.100/24",
		},
		Groups: map[string][]string{
			"group:example": {
				"user1@example.com",
				"user2@example.com",
			},
		},
		Tests: []tailscale.ACLTest{
			{
				User:  "user1@example.com",
				Allow: []string{"example-host-1:22", "example-host-2:80"},
				Deny:  []string{"exapmle-host-2:100"},
			},
			{
				User:  "user2@example.com",
				Allow: []string{"100.60.3.4:22"},
			},
		},
	}

	acl, err := client.ACL(context.Background())
	assert.NoError(t, err)
	assert.EqualValues(t, server.ResponseBody, acl)
	assert.EqualValues(t, http.MethodGet, server.Method)
	assert.EqualValues(t, "/api/v2/tailnet/example.com/acl", server.Path)
}

func TestClient_SetDeviceSubnetRoutes(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	const deviceID = "test"
	routes := []string{"127.0.0.1"}

	assert.NoError(t, client.SetDeviceSubnetRoutes(context.Background(), deviceID, routes))
	assert.Equal(t, http.MethodPost, server.Method)
	assert.Equal(t, "/api/v2/device/test/routes", server.Path)

	body := make(map[string][]string)
	assert.NoError(t, json.Unmarshal(server.Body.Bytes(), &body))
	assert.EqualValues(t, routes, body["routes"])
}

func TestClient_Devices(t *testing.T) {
	t.Parallel()

	expectedDevices := map[string][]tailscale.Device{
		"devices": {
			{
				Name: "test",
				ID:   "test",
			},
		},
	}

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK
	server.ResponseBody = expectedDevices

	actualDevices, err := client.Devices(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/devices", server.Path)
	assert.EqualValues(t, expectedDevices["devices"], actualDevices)

}

func TestClient_DeviceSubnetRoutes(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK
	server.ResponseBody = &tailscale.DeviceRoutes{
		Advertised: []string{"127.0.0.1"},
		Enabled:    []string{"127.0.0.1"},
	}

	const deviceID = "test"

	routes, err := client.DeviceSubnetRoutes(context.Background(), deviceID)
	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, server.Method)
	assert.Equal(t, "/api/v2/device/test/routes", server.Path)
	assert.Equal(t, server.ResponseBody, routes)
}

func TestClient_DNSNameservers(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	expectedNameservers := map[string][]string{
		"dns": {"127.0.0.1"},
	}

	server.ResponseBody = expectedNameservers
	nameservers, err := client.DNSNameservers(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/dns/nameservers", server.Path)
	assert.Equal(t, expectedNameservers["dns"], nameservers)
}

func TestClient_DNSPreferences(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK
	server.ResponseBody = &tailscale.DNSPreferences{
		MagicDNS: true,
	}

	preferences, err := client.DNSPreferences(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/dns/preferences", server.Path)
	assert.Equal(t, server.ResponseBody, preferences)
}

func TestClient_DNSSearchPaths(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	expectedPaths := map[string][]string{
		"searchPaths": {"test"},
	}

	server.ResponseBody = expectedPaths

	paths, err := client.DNSSearchPaths(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/dns/searchpaths", server.Path)
	assert.Equal(t, expectedPaths["searchPaths"], paths)
}

func TestClient_SetDNSNameservers(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	nameservers := []string{"127.0.0.1"}

	assert.NoError(t, client.SetDNSNameservers(context.Background(), nameservers))
	assert.Equal(t, http.MethodPost, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/dns/nameservers", server.Path)

	body := make(map[string][]string)
	assert.NoError(t, json.Unmarshal(server.Body.Bytes(), &body))
	assert.EqualValues(t, nameservers, body["dns"])
}

func TestClient_SetDNSPreferences(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	preferences := tailscale.DNSPreferences{
		MagicDNS: true,
	}

	assert.NoError(t, client.SetDNSPreferences(context.Background(), preferences))
	assert.Equal(t, http.MethodPost, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/dns/preferences", server.Path)

	var body tailscale.DNSPreferences
	assert.NoError(t, json.Unmarshal(server.Body.Bytes(), &body))
	assert.EqualValues(t, preferences, body)
}

func TestClient_SetDNSSearchPaths(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	paths := []string{"test"}

	assert.NoError(t, client.SetDNSSearchPaths(context.Background(), paths))
	assert.Equal(t, http.MethodPost, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/dns/searchpaths", server.Path)

	body := make(map[string][]string)
	assert.NoError(t, json.Unmarshal(server.Body.Bytes(), &body))
	assert.EqualValues(t, paths, body["searchPaths"])
}

func TestClient_AuthorizeDevice(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	const deviceID = "test"

	assert.NoError(t, client.AuthorizeDevice(context.Background(), deviceID))
	assert.Equal(t, http.MethodPost, server.Method)
	assert.Equal(t, "/api/v2/device/test/authorized", server.Path)

	body := make(map[string]bool)
	assert.NoError(t, json.Unmarshal(server.Body.Bytes(), &body))
	assert.EqualValues(t, true, body["authorized"])
}

func TestClient_CreateKey(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	capabilities := tailscale.KeyCapabilities{}
	capabilities.Devices.Create.Ephemeral = true
	capabilities.Devices.Create.Reusable = true

	expected := tailscale.Key{
		ID:           "test",
		Key:          "thisisatestkey",
		Created:      time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		Expires:      time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		Capabilities: capabilities,
	}

	server.ResponseBody = expected

	actual, err := client.CreateKey(context.Background(), capabilities)
	assert.NoError(t, err)
	assert.EqualValues(t, expected, actual)
	assert.Equal(t, http.MethodPost, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/keys", server.Path)
}

func TestClient_GetKey(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	capabilities := tailscale.KeyCapabilities{}
	capabilities.Devices.Create.Ephemeral = true
	capabilities.Devices.Create.Reusable = true

	expected := tailscale.Key{
		ID:           "test",
		Created:      time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		Expires:      time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		Capabilities: capabilities,
	}

	server.ResponseBody = expected

	actual, err := client.GetKey(context.Background(), expected.ID)
	assert.NoError(t, err)
	assert.EqualValues(t, expected, actual)
	assert.Equal(t, http.MethodGet, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/keys/"+expected.ID, server.Path)
}

func TestClient_DeleteKey(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusOK

	const keyID = "test"

	assert.NoError(t, client.DeleteKey(context.Background(), keyID))
	assert.Equal(t, http.MethodDelete, server.Method)
	assert.Equal(t, "/api/v2/tailnet/example.com/keys/"+keyID, server.Path)
}

func TestIsNotFound(t *testing.T) {
	t.Parallel()

	client, server := NewTestHarness(t)
	server.ResponseCode = http.StatusNotFound
	server.ResponseBody = tailscale.APIError{Message: "error"}

	_, err := client.GetKey(context.Background(), "test")
	assert.True(t, tailscale.IsNotFound(err))
}
