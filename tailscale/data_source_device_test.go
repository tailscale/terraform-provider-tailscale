// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	tsclient "tailscale.com/client/tailscale/v2"
	"tailscale.com/tstest"
)

func TestDeviceToMap(t *testing.T) {
	t.Parallel()
	cl := tstest.NewClock(tstest.ClockOpts{})
	created := tsclient.Time{Time: cl.Now().Truncate(time.Second)}
	expires := tsclient.Time{Time: cl.Now().Truncate(time.Second).Add(24 * time.Hour)}
	lastSeen := tsclient.Time{Time: cl.Now().Truncate(time.Second).Add(-5 * time.Minute)}

	dev := &tsclient.Device{
		Name:                      "host.example.ts.net",
		Hostname:                  "host",
		User:                      "user@example.com",
		NodeID:                    "node-123",
		Addresses:                 []string{"100.100.100.101", "fd7a:115c:a1e0::1"},
		Tags:                      []string{"tag:test1", "tag:test2"},
		Authorized:                true,
		KeyExpiryDisabled:         true,
		BlocksIncomingConnections: true,
		ClientVersion:             "1.88.4",
		Created:                   created,
		Expires:                   expires,
		IsExternal:                false,
		LastSeen:                  &lastSeen,
		MachineKey:                "machine-key",
		NodeKey:                   "node-key",
		OS:                        "linux",
		UpdateAvailable:           true,
		TailnetLockError:          "lock-error",
		TailnetLockKey:            "lock-key",
	}

	m := deviceToMap(dev)

	assert.Equal(t, dev.Name, m["name"].(string))
	assert.Equal(t, dev.Hostname, m["hostname"].(string))
	assert.Equal(t, dev.User, m["user"].(string))
	assert.Equal(t, dev.NodeID, m["node_id"].(string))
	assert.Equal(t, dev.Addresses, m["addresses"].([]string))
	assert.Equal(t, dev.Tags, m["tags"].([]string))
	assert.Equal(t, dev.Authorized, m["authorized"].(bool))
	assert.Equal(t, dev.KeyExpiryDisabled, m["key_expiry_disabled"].(bool))
	assert.Equal(t, dev.BlocksIncomingConnections, m["blocks_incoming_connections"].(bool))
	assert.Equal(t, dev.ClientVersion, m["client_version"].(string))
	assert.Equal(t, created.Format(time.RFC3339), m["created"].(string))
	assert.Equal(t, expires.Format(time.RFC3339), m["expires"].(string))
	assert.Equal(t, dev.IsExternal, m["is_external"].(bool))
	assert.Equal(t, lastSeen.Format(time.RFC3339), m["last_seen"].(string))
	assert.Equal(t, dev.MachineKey, m["machine_key"].(string))
	assert.Equal(t, dev.NodeKey, m["node_key"].(string))
	assert.Equal(t, dev.OS, m["os"].(string))
	assert.Equal(t, dev.UpdateAvailable, m["update_available"].(bool))
	assert.Equal(t, dev.TailnetLockError, m["tailnet_lock_error"].(string))
	assert.Equal(t, dev.TailnetLockKey, m["tailnet_lock_key"].(string))
}
func TestDeviceToMap_LastSeenNil(t *testing.T) {
	t.Parallel()
	cl := tstest.NewClock(tstest.ClockOpts{})
	created := tsclient.Time{Time: cl.Now().Truncate(time.Second)}
	expires := tsclient.Time{Time: cl.Now().Truncate(time.Second).Add(24 * time.Hour)}

	dev := &tsclient.Device{
		Name:                      "host.example.ts.net",
		Hostname:                  "host",
		User:                      "user@example.com",
		NodeID:                    "node-123",
		Addresses:                 []string{"100.100.100.101", "fd7a:115c:a1e0::1"},
		Tags:                      []string{"tag:test1", "tag:test2"},
		Authorized:                true,
		KeyExpiryDisabled:         true,
		BlocksIncomingConnections: true,
		ClientVersion:             "1.88.4",
		Created:                   created,
		Expires:                   expires,
		IsExternal:                false,
		LastSeen:                  nil,
		MachineKey:                "machine-key",
		NodeKey:                   "node-key",
		OS:                        "linux",
		UpdateAvailable:           true,
		TailnetLockError:          "lock-error",
		TailnetLockKey:            "lock-key",
	}

	m := deviceToMap(dev)

	assert.Equal(t, dev.Name, m["name"].(string))
	assert.Equal(t, dev.Hostname, m["hostname"].(string))
	assert.Equal(t, dev.User, m["user"].(string))
	assert.Equal(t, dev.NodeID, m["node_id"].(string))
	assert.Equal(t, dev.Addresses, m["addresses"].([]string))
	assert.Equal(t, dev.Tags, m["tags"].([]string))
	assert.Equal(t, dev.Authorized, m["authorized"].(bool))
	assert.Equal(t, dev.KeyExpiryDisabled, m["key_expiry_disabled"].(bool))
	assert.Equal(t, dev.BlocksIncomingConnections, m["blocks_incoming_connections"].(bool))
	assert.Equal(t, dev.ClientVersion, m["client_version"].(string))
	assert.Equal(t, created.Format(time.RFC3339), m["created"].(string))
	assert.Equal(t, expires.Format(time.RFC3339), m["expires"].(string))
	assert.Equal(t, dev.IsExternal, m["is_external"].(bool))
	assert.Equal(t, "", m["last_seen"]) // Expect empty string for nil LastSeen
	assert.Equal(t, dev.MachineKey, m["machine_key"].(string))
	assert.Equal(t, dev.NodeKey, m["node_key"].(string))
	assert.Equal(t, dev.OS, m["os"].(string))
	assert.Equal(t, dev.UpdateAvailable, m["update_available"].(bool))
	assert.Equal(t, dev.TailnetLockError, m["tailnet_lock_error"].(string))
	assert.Equal(t, dev.TailnetLockKey, m["tailnet_lock_key"].(string))
}
