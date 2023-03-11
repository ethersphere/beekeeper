package mocks

import "net/http"

// MockRoundTripper is a mock implementation of the RoundTripper interface.
type MockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

// RoundTrip calls the RoundTripFunc method of the mock.
func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}
