package api

import "testing"

func TestGetRole(t *testing.T) {
	tt := []struct {
		desc         string
		path, method string
		expectedRole string
	}{
		{desc: "plain", expectedRole: "role1", path: "/bytes", method: "POST"},
		{desc: "query param", expectedRole: "role1", path: "/v1/bzz?name=settlements-2", method: "POST"},
		{desc: "multi method 1", expectedRole: "role1", path: "/tags", method: "POST"},
		{desc: "multi method 2", expectedRole: "role1", path: "/tags", method: "GET"},
		{desc: "one level", expectedRole: "role0", path: "/bytes/123", method: "GET"},
		{desc: "two levels", expectedRole: "role2", path: "/stamps/1/17", method: "POST"},
	}
	for _, tc := range tt {
		t.Run(tc.desc, func(t *testing.T) {
			if got := getRole(tc.path, tc.method); got != tc.expectedRole {
				t.Errorf("expected %s, got %s", tc.expectedRole, got)
			}
		})
	}
}
