// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ planmodifier.String = jsonSemanticDiffModifier{}
	_ planmodifier.String = PreserveEmptyStringAsNull{}
)

// jsonSemanticDiffModifier is a plan modifier that will treat strings as
// equivalent if they correspond to the same JSON value, and differ only
// in whitespace.
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

// PreserveEmptyStringAsNull is a plan modifier that will treat empty strings in
// the state as equivalent to null values, and not change them. This is needed
// because the plugin SDK provider may have saved empty strings in the state for
// certain attributes when set to null, but the plugin framework-based provider
// will always save null strings as null. In cases where the empty string and
// null are equivalent as far as the client or API are concerned, we therefore
// need to change the plan to avoid changing an empty string to a null, and a
// confusing no-op diff from Terraform. For more details, see:
//   - https://github.com/hashicorp/terraform-plugin-framework/issues/510
//   - https://discuss.hashicorp.com/t/framework-migration-test-produces-non-empty-plan/54523/12
type PreserveEmptyStringAsNull struct{}

func (pm PreserveEmptyStringAsNull) Description(_ context.Context) string {
	return `If the existing value of this attribute in state is "" and the new value is null, the value of this attribute in state will remain as the empty string.`
}

func (pm PreserveEmptyStringAsNull) MarkdownDescription(_ context.Context) string {
	return `If the existing value of this attribute in state is "" and the new value is null, the value of this attribute in state will remain as the empty string.`
}

func (pm PreserveEmptyStringAsNull) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.ValueString() == "" && !req.StateValue.IsUnknown() && req.ConfigValue.IsNull() {
		resp.PlanValue = types.StringValue("")
	}
}
