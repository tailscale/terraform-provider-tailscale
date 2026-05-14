// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/tailscale/hujson"
)

var (
	_ validator.String = cidrValidator{}
	_ validator.String = retryDeadlineValidator{}
	_ validator.String = aclHuJSONValidator{}
	_ validator.List   = atLeastOneBlockRequiredValidator{}
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

// aclHuJSONValidator is a [validator.String] that checks whether a string can be
// parsed as HuJSON.
type aclHuJSONValidator struct{}

func (v aclHuJSONValidator) Description(_ context.Context) string {
	return "string must be a valid HuJSON or JSON document"
}

func (v aclHuJSONValidator) MarkdownDescription(_ context.Context) string {
	return "string must be a valid **HuJSON** or **JSON** document"
}

func (v aclHuJSONValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	if _, err := hujson.Parse([]byte(req.ConfigValue.ValueString())); err != nil {
		resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
			req.Path,
			"is invalid HuJSON",
			err.Error(),
		))
	}
}

// atLeastOneBlockRequiredValidator validates that a list has a configuration
// value. Intended for use with `schema.ListNestedBlock`.
type atLeastOneBlockRequiredValidator struct{}

// Description describes the validation in plain text formatting.
func (v atLeastOneBlockRequiredValidator) Description(_ context.Context) string {
	return "must have at least one nested block configured"
}

// MarkdownDescription describes the validation in Markdown formatting.
func (v atLeastOneBlockRequiredValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// Validate performs the validation.
func (v atLeastOneBlockRequiredValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	len := req.ConfigValue.Length(basetypes.CollectionLengthOptions{
		UnhandledNullAsZero:    true,
		UnhandledUnknownAsZero: true,
	})
	if len == 0 {
		last, _ := req.Path.Steps().LastStep()
		resp.Diagnostics.AddAttributeError(req.Path,
			fmt.Sprintf("Insufficient %s blocks", last),
			fmt.Sprintf("At least 1 %q blocks are required.", last))
	}
}

// AtLeastOneBlockRequired returns a validator which ensures that at least one
// instance of the nested block is configured.
//
// This validator is similar to the `Required` field on attributes and is only
// practical for use with `schema.ListNestedBlock`.
//
// Unlike [listvalidator.SizeAtLeast] it does not ignore null or unknown values.
func AtLeastOneBlockRequired() validator.List {
	return atLeastOneBlockRequiredValidator{}
}
