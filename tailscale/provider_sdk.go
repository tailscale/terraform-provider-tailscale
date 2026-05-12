// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

// Package tailscale describes the resources and data sources provided by the terraform provider. Each resource
// or data source is described within its own file.
package tailscale

import (
	"errors"
	"os"

	"github.com/hashicorp/go-uuid"
)

// providerVersion is filled by goreleaser at build time.
var providerVersion = "dev"

func validateProviderCreds(apiKey, oauthClientID, oauthClientSecret, idToken, audience string) error {
	if apiKey == "" && oauthClientID == "" && oauthClientSecret == "" && idToken == "" && audience == "" {
		return errors.New("tailscale provider credentials are empty - set `api_key` or 'oauth_client_id' and one of 'oauth_client_secret', 'identity_token', or 'audience'")
	} else if apiKey != "" && (oauthClientID != "" || oauthClientSecret != "" || idToken != "" || audience != "") {
		return errors.New("tailscale provider credentials are conflicting - `api_key` conflicts with 'oauth_client_id', 'oauth_client_secret', 'identity_token', and 'audience'")
	} else if audience != "" && (oauthClientSecret != "" || idToken != "") {
		return errors.New("tailscale provider argument 'audience' conflicts with 'oauth_client_secret' and 'identity_token'")
	} else if apiKey == "" && oauthClientID == "" {
		return errors.New("tailscale provider argument 'oauth_client_id' is empty")
	} else if oauthClientID != "" && oauthClientSecret == "" && idToken == "" && audience == "" {
		return errors.New("one of tailscale provider arguments 'oauth_client_secret', 'identity_token', or 'audience' are mandatory with 'oauth_client_id'")
	}

	return nil
}

func createUUID() string {
	val, err := uuid.GenerateUUID()
	if err != nil {
		panic(err)
	}
	return val
}

// isAcceptanceTesting returns true if we're running acceptance tests.
func isAcceptanceTesting() bool {
	return os.Getenv("TF_ACC") != ""
}
