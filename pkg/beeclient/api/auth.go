package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// AuthService represents Bee's Auth service
type AuthService service

// AuthResponse represents authentication response
type AuthResponse struct {
	Key string `json:"key"`
}

const roleTmpl = `{
    "role": "%s"
}`

// Authenticate gets the bearer security token based on given credentials
func (b *AuthService) Authenticate(ctx context.Context, role, username, password string) (resp AuthResponse, err error) {
	plain := fmt.Sprintf("%s:%s", username, password)
	encoded := base64.StdEncoding.EncodeToString([]byte(plain))

	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("Accept", "application/json")
	header.Set("Authorization", "Basic "+encoded)

	data := strings.NewReader(fmt.Sprintf(roleTmpl, role))

	req, err := http.NewRequest(http.MethodPost, "/auth", data)
	if err != nil {
		return AuthResponse{}, err
	}

	req = req.WithContext(ctx)
	req.Header = header

	r, err := b.client.httpClient.Do(req)
	if err != nil {
		return AuthResponse{}, err
	}

	if err = responseErrorHandler(r); err != nil {
		return AuthResponse{}, err
	}

	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err = json.NewDecoder(r.Body).Decode(&resp); err != nil && err != io.EOF {
			return AuthResponse{}, err
		}
	}

	return
}
