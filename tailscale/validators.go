// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"net"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	_ validator.String = cidrValidator{}
	_ validator.String = retryDeadlineValidator{}
)

// cidrValidator is a [validator.String] for CIDR addresses.
type cidrValidator struct{}

func (v cidrValidator) Description(_ context.Context) string {
	return "value must be a CIDR address"
}

func (v cidrValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v cidrValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()
	_, _, err := net.ParseCIDR(value)
	if err != nil {
		resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
			req.Path,
			v.Description(ctx),
			req.ConfigValue.ValueString(),
		))
	}
}

// retryDeadlineValdiator is a [validator.String] that checks whether a string can be
// parsed as a duration greater than 1s.
type retryDeadlineValidator struct{}

func (r retryDeadlineValidator) Description(_ context.Context) string {
	return "Validates that the value is a duration greater than 1s."
}

func (r retryDeadlineValidator) MarkdownDescription(ctx context.Context) string {
	return r.Description(ctx)
}

func (r retryDeadlineValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() {
		return
	}
	waitFor, err := time.ParseDuration(req.ConfigValue.ValueString())
	switch {
	case err != nil:
		resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
			req.Path,
			"unable to parse value as a duration",
			req.ConfigValue.ValueString(),
		))
	case waitFor <= time.Second:
		resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
			req.Path,
			"duration must be greater than 1 second",
			req.ConfigValue.ValueString(),
		))
	default:
	}
}
