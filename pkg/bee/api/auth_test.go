package api

import "testing"

func TestGetRole(t *testing.T) {
	tt := []struct {
		desc         string
		path, method string
		expectedRole string
	}{
		{desc: "plain", expectedRole: "creator", path: "/bytes", method: "POST"},
		{desc: "query param", expectedRole: "creator", path: "/v1/bzz?name=settlements-2", method: "POST"},
		{desc: "multi method 1", expectedRole: "creator", path: "/tags", method: "POST"},
		{desc: "multi method 2", expectedRole: "creator", path: "/tags", method: "GET"},
		{desc: "one level", expectedRole: "consumer", path: "/bytes/123", method: "GET"},
		{desc: "two levels", expectedRole: "maintainer", path: "/stamps/1/17", method: "POST"},
	}
	for _, tc := range tt {
		t.Run(tc.desc, func(t *testing.T) {
			if got := getRole(tc.path, tc.method); got != tc.expectedRole {
				t.Errorf("expected %s, got %s", tc.expectedRole, got)
			}
		})
	}
}
