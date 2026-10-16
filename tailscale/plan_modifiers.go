// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/tailscale/hujson"
)

var (
	_ planmodifier.String = jsonSemanticDiffModifier{}
	_ planmodifier.String = aclHuJSONModifier{}
	_ planmodifier.Bool   = boolDefaultIfUnknownModifier{}
)

// BoolDefaultIfUnknown returns a plan modifier that resolves an unknown planned value to
// value. It's intended for use on a [schema.BoolAttribute] that is both Optional
// and Computed. It only considers values to be unknown if they are neither set
// in the configuration nor on the remote resource.
//
// This means that removing the attribute from a configuration leaves the last
// applied value in place instead of reverting to value. A value the
// configuration sets from an expression that is still unresolved is also left
// alone, so it is not replaced by value before Terraform has learnt it.
func BoolDefaultIfUnknown(value bool) planmodifier.Bool {
	return boolDefaultIfUnknownModifier{value: value}
}

// boolDefaultIfUnknownModifier resolves an unknown planned value to a fixed value.
type boolDefaultIfUnknownModifier struct {
	value bool
}

func (m boolDefaultIfUnknownModifier) Description(_ context.Context) string {
	return fmt.Sprintf("Defaults to %t.", m.value)
}

func (m boolDefaultIfUnknownModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m boolDefaultIfUnknownModifier) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	// Only ever apply a default for a value that the configuration
	// leaves empty.
	if req.PlanValue.IsUnknown() && req.ConfigValue.IsNull() {
		resp.PlanValue = types.BoolValue(m.value)
	}
}

// jsonSemanticDiffModifier treats strings as equivalent if they correspond
// to the same JSON value, and differ only in whitespace.
type jsonSemanticDiffModifier struct{}

func (m jsonSemanticDiffModifier) Description(_ context.Context) string {
	return "Suppresses diffs if JSON is semantically equal"
}

func (m jsonSemanticDiffModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m jsonSemanticDiffModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.ConfigValue.IsNull() || req.StateValue.IsNull() {
		return
	}

	var config, state map[string]any
	if err := json.Unmarshal([]byte(req.ConfigValue.ValueString()), &config); err != nil {
		return
	}
	if err := json.Unmarshal([]byte(req.StateValue.ValueString()), &state); err != nil {
		return
	}

	if reflect.DeepEqual(config, state) {
		resp.PlanValue = req.StateValue
	}
}

// aclHuJSONModifier stores strings as their canonical HuJSON representation in
// the state, and treats strings as equivalent if their canonicalical representation
// is the same.
type aclHuJSONModifier struct{}

func (m aclHuJSONModifier) Description(_ context.Context) string {
	return "Suppresses diffs if two strings have the same canonical HuJSON value."
}

func (m aclHuJSONModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m aclHuJSONModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	formatted, err := hujson.Format([]byte(req.ConfigValue.ValueString()))
	if err != nil {
		return
	}

	normalizedConfig := string(formatted)

	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
		stateFormatted, err := hujson.Format([]byte(req.StateValue.ValueString()))
		if err == nil && string(stateFormatted) == normalizedConfig {
			resp.PlanValue = req.StateValue
			return
		}
	}
}
