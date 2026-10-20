// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/tailscale/hujson"
)

var (
	_ basetypes.StringTypable                    = aclHuJSONType{}
	_ basetypes.StringValuableWithSemanticEquals = aclHuJSONValue{}
)

type aclHuJSONType struct {
	basetypes.StringType
}

func (t aclHuJSONType) Equal(other attr.Type) bool {
	otherType, ok := other.(aclHuJSONType)
	return ok && t.StringType.Equal(otherType.StringType)
}

func (t aclHuJSONType) String() string {
	return "tailscale.aclHuJSONType"
}

func (t aclHuJSONType) ValueFromString(_ context.Context, value basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return aclHuJSONValue{StringValue: value}, nil
}

func (t aclHuJSONType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.StringType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}

	stringValue, ok := converted.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T", converted)
	}
	return aclHuJSONValue{StringValue: stringValue}, nil
}

func (t aclHuJSONType) ValueType(context.Context) attr.Value {
	return aclHuJSONValue{}
}

type aclHuJSONValue struct {
	basetypes.StringValue
}

func newACLHuJSONValue(value string) aclHuJSONValue {
	return aclHuJSONValue{StringValue: basetypes.NewStringValue(value)}
}

func (v aclHuJSONValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(aclHuJSONValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(otherValue.StringValue)
}

func (v aclHuJSONValue) Type(context.Context) attr.Type {
	return aclHuJSONType{}
}

// StringSemanticEquals lets the framework retain prior user formatting when
// the API returns an equivalent canonical HuJSON value during refresh.
func (v aclHuJSONValue) StringSemanticEquals(_ context.Context, other basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	otherValue, ok := other.(aclHuJSONValue)
	if !ok {
		diags.AddError(
			"HuJSON Semantic Equality Check Error",
			fmt.Sprintf("expected %T, got %T", v, other),
		)
		return false, diags
	}

	currentFormatted, currentErr := hujson.Format([]byte(v.ValueString()))
	otherFormatted, otherErr := hujson.Format([]byte(otherValue.ValueString()))
	if currentErr != nil || otherErr != nil {
		diags.AddError(
			"HuJSON Semantic Equality Check Error",
			fmt.Sprintf("could not format values for comparison: current error: %v; new error: %v", currentErr, otherErr),
		)
		return false, diags
	}

	return string(currentFormatted) == string(otherFormatted), diags
}
