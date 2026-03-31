package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

type Pebble struct {
	httpClient *http.Client
}

func NewPebble(httpClient *http.Client) *Pebble {
	return &Pebble{
		httpClient: httpClient,
	}
}

func (p *Pebble) FetchRootCA(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s returned %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	return string(body), nil
}
