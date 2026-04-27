// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

var (
	_ planmodifier.String = jsonSemanticDiffModifier{}
)

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
