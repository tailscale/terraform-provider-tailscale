// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCIDRValidator(t *testing.T) {
	testCases := []stringValidatorTestCase{
		{
			name:   "valid-cidr",
			config: types.StringValue("1.2.3.4/28"),
		},
		{
			name:    "invalid-value",
			config:  types.StringValue("1234567890"),
			wantErr: true,
		},
		{
			name:    "no-slash",
			config:  types.StringValue("1.2.3.4"),
			wantErr: true,
		},
		{
			name:    "empty",
			config:  types.StringValue(""),
			wantErr: true,
		},
	}

	runStringValidatorTests(t, cidrValidator{}, testCases)
}

func TestRetryDeadlineValidator(t *testing.T) {
	testCases := []stringValidatorTestCase{
		{
			name:   "valid-duration",
			config: types.StringValue("5s"),
		},
		{
			name:    "invalid-duration",
			config:  types.StringValue("abc"),
			wantErr: true,
		},
		{
			name:    "duration-1s",
			config:  types.StringValue("1s"),
			wantErr: true,
		},
		{
			name:    "duration-lt-1s",
			config:  types.StringValue("1ms"),
			wantErr: true,
		},
	}

	runStringValidatorTests(t, retryDeadlineValidator{}, testCases)
}

func TestAclHuJSONValidator(t *testing.T) {
	testCases := []stringValidatorTestCase{
		{
			name:   "valid-json",
			config: types.StringValue(`{ "grants": [ {"src": ["*"], "dst": ["*"], "ip": ["*"] } ] }`),
		},
		{
			name: "valid-hujson-with-comments",
			config: types.StringValue(`{
				// Allow all connections. 
				"grants": [ {"src": ["*"], "dst": ["*"], "ip": ["*"] } ],
			}`),
		},
		{
			name:    "invalid-json",
			config:  types.StringValue("{ //"),
			wantErr: true,
		},
		{
			name:   "null",
			config: types.StringNull(),
		},
		{
			name:   "unknown",
			config: types.StringUnknown(),
		},
	}

	runStringValidatorTests(t, aclHuJSONValidator{}, testCases)
}

type stringValidatorTestCase struct {
	name    string
	config  types.String
	wantErr bool
}

// runStringValidatorTests goes through the test cases, applies the
// validator, and checks if the string is passed/errored as expected.
func runStringValidatorTests(t *testing.T, stringValidator validator.String, testCases []stringValidatorTestCase) {
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: tt.config,
			}

			resp := validator.StringResponse{
				Diagnostics: diag.Diagnostics{},
			}

			t.Run(tt.name, func(t *testing.T) {
				stringValidator.ValidateString(t.Context(), req, &resp)

				hasError := resp.Diagnostics.HasError()
				if hasError && !tt.wantErr {
					t.Errorf("got unexpected error from validator: %v", resp.Diagnostics.Errors())
				} else if !hasError && tt.wantErr {
					t.Errorf("validator passed, expected failure")
				}
			})
		})
	}
}
