// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"

	"github.com/tailscale/hujson"
)

var (
	_ planmodifier.String = jsonSemanticDiffModifier{}
	_ planmodifier.String = aclHuJSONModifier{}
)

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
