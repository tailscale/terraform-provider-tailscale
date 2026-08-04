// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import "testing"

func TestAclHuJSONValueSemanticEquals(t *testing.T) {
	testCases := []struct {
		name     string
		current  string
		newValue string
		equal    bool
	}{
		{
			name: "formatting-only difference",
			current: `{
	"hosts": {
		"example": "100.64.0.1",
	},
}
`,
			newValue: `{
  "hosts": {
    "example": "100.64.0.1",
  },
}`,
			equal: true,
		},
		{
			name:     "semantic difference",
			current:  `{"hosts":{"example":"100.64.0.1"}}`,
			newValue: `{"hosts":{"example":"100.64.0.2"}}`,
			equal:    false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			current := newACLHuJSONValue(tt.current)
			newValue := newACLHuJSONValue(tt.newValue)

			equal, diags := current.StringSemanticEquals(t.Context(), newValue)
			if diags.HasError() {
				t.Fatalf("semantic equality returned errors: %v", diags)
			}
			if equal != tt.equal {
				t.Fatalf("semantic equality = %v, want %v", equal, tt.equal)
			}
		})
	}
}
