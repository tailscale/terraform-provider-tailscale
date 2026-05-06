// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestJsonSemanticDiffModifier(t *testing.T) {
	testCases := []stringPlanModifierTestCase{
		{
			name:         "empty-objects",
			state:        types.StringValue(`{}`),
			config:       types.StringValue(`{}`),
			expectedPlan: types.StringValue(`{}`),
		},
		{
			name: "equivalent-json",
			state: types.StringValue(`{
							"sides": 5,
							"colour": "blue"
						}`),
			config: types.StringValue(`{"sides":5,"colour":"blue"}`),
			expectedPlan: types.StringValue(`{
							"sides": 5,
							"colour": "blue"
						}`),
		},
		{
			name:         "different-json",
			state:        types.StringValue(`{ "sides": 4, "colour": "green" }`),
			config:       types.StringValue(`["apple", "banana", "cherry"]`),
			expectedPlan: types.StringValue(`["apple", "banana", "cherry"]`),
		},
		{
			name:         "non-json-in-state",
			state:        types.StringValue("<xml>this isn't JSON</xml>"),
			config:       types.StringValue(`{"sides": 6, "colour": "yellow"}`),
			expectedPlan: types.StringValue(`{"sides": 6, "colour": "yellow"}`),
		},
		{
			name:         "non-json-in-plan",
			state:        types.StringValue(`{"sides": 7, "colour": "orange"}`),
			config:       types.StringValue("<xml>this isn't JSON either</xml>"),
			expectedPlan: types.StringValue("<xml>this isn't JSON either</xml>"),
		},
		{
			name:         "config-null",
			state:        types.StringValue(`{"sides": 8, "colour": "red"}`),
			config:       types.StringNull(),
			expectedPlan: types.StringNull(),
		},
		{
			name:         "config-unknown",
			state:        types.StringValue(`{"sides": 1, "colour": "pink"}`),
			config:       types.StringUnknown(),
			expectedPlan: types.StringUnknown(),
		},
	}

	runStringPlanModifierTests(t, jsonSemanticDiffModifier{}, testCases)
}

func TestAclHuJSONModifier(t *testing.T) {
	testCases := []stringPlanModifierTestCase{
		{
			name:         "empty-objects",
			state:        types.StringValue(`{}`),
			config:       types.StringValue(`{}`),
			expectedPlan: types.StringValue(`{}`),
		},
		{
			name:         "equivalent-json",
			state:        types.StringValue(`{"wheels": 3, "seats": 2}`),
			config:       types.StringValue(`{"wheels": 3, "seats": 2}`),
			expectedPlan: types.StringValue(`{"wheels": 3, "seats": 2}`),
		},
		{
			name: "unformatted-plan-is-left-as-is",
			config: types.StringValue(`{
				"wheels": 6,
					"seats": 8
			}`),
			expectedPlan: types.StringValue(`{
				"wheels": 6,
					"seats": 8
			}`),
		},
		{
			// This assumes the Terraform state contains non-canonicalised HuJSON,
			// which seems unlikely, but check it would produce an empty diff just in case.
			name: "plan-matches-state-if-equivalent",
			state: types.StringValue(`{
				"wheels": 6,
					"seats": 8
			}`),
			config: types.StringValue("{\n\t\"wheels\": 6,\n\t\"seats\": 8\n}\n"),
			expectedPlan: types.StringValue(`{
				"wheels": 6,
					"seats": 8
			}`),
		},
		{
			name:         "config-invalid-json-is-left-as-is",
			state:        types.StringUnknown(),
			config:       types.StringValue("<xml>not json</xml>"),
			expectedPlan: types.StringValue("<xml>not json</xml>"),
		},
		{
			name:         "config-null",
			state:        types.StringValue(`{"wheels": 4, "seats": 5}`),
			config:       types.StringNull(),
			expectedPlan: types.StringNull(),
		},
		{
			name:         "config-unknown",
			state:        types.StringValue(`{"wheels": 1, "seats": 1}`),
			config:       types.StringUnknown(),
			expectedPlan: types.StringUnknown(),
		},
	}

	runStringPlanModifierTests(t, aclHuJSONModifier{}, testCases)
}

func TestPreserveEmptyStringAsNull(t *testing.T) {
	testCases := []stringPlanModifierTestCase{
		{
			name:         "equal-strings",
			state:        types.StringValue("cabbage"),
			config:       types.StringValue("cabbage"),
			expectedPlan: types.StringValue("cabbage"),
		},
		{
			name:         "different-strings",
			state:        types.StringValue("carrot"),
			config:       types.StringValue("lettuce"),
			expectedPlan: types.StringValue("lettuce"),
		},
		{
			name:         "empty-string-in-state-non-empty-in-plan",
			state:        types.StringValue(""),
			config:       types.StringValue("broccoli"),
			expectedPlan: types.StringValue("broccoli"),
		},
		{
			name:         "non-empty-string-in-state-empty-in-plan",
			state:        types.StringValue("rhubarb"),
			config:       types.StringValue(""),
			expectedPlan: types.StringValue(""),
		},
		{
			name:         "empty-string-in-state-unknown-plan",
			state:        types.StringValue(""),
			config:       types.StringUnknown(),
			expectedPlan: types.StringUnknown(),
		},
		{
			name:         "empty-string-in-state-null-plan",
			state:        types.StringValue(""),
			config:       types.StringNull(),
			expectedPlan: types.StringValue(""),
		},
		{
			name:         "null-in-state-empty-string-in-plan",
			state:        types.StringNull(),
			config:       types.StringValue(""),
			expectedPlan: types.StringValue(""),
		},
	}

	runStringPlanModifierTests(t, PreserveEmptyStringAsNull{}, testCases)
}

type stringPlanModifierTestCase struct {
	name         string
	state        types.String
	config       types.String
	expectedPlan types.String
}

// runStringPlanModifierTests goes through the test cases, applies the
// plan modifier, and checks if the PlanValue matches the expected.
func runStringPlanModifierTests(t *testing.T, modifier planmodifier.String, testCases []stringPlanModifierTestCase) {
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// StateValue is the value currently stored in the state,
			// ConfigValue is the exact value written by the user in their configuration,
			// PlanValue is the value Terraform proposes to save
			//
			// Initially the PlanValue is the same as the ConfigValue, and we check
			// if it's been updated correctly (or left as-is) after we've applied
			// the plan modifier.
			req := planmodifier.StringRequest{
				StateValue:  tt.state,
				ConfigValue: tt.config,
				PlanValue:   tt.config,
			}
			resp := planmodifier.StringResponse{
				PlanValue: req.PlanValue,
			}

			modifier.PlanModifyString(t.Context(), req, &resp)

			if resp.PlanValue != tt.expectedPlan {
				t.Errorf("plan value is incorrect: got %s, want %s", resp.PlanValue, tt.expectedPlan)
			}
		})
	}
}
