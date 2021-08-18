package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

// AuthService represents Bee's Auth service
type AuthService service

const roleTmpl = `{
    "role": "%s"
}`

// Authenticate gets the bearer security token based on given credentials
func (b *AuthService) Authenticate(ctx context.Context, role, username, password string) (resp AuthResponse, err error) {
	plain := fmt.Sprintf("%s:%s", username, password)
	encoded := base64.StdEncoding.EncodeToString([]byte(plain))

	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("Authorization", "Basic "+encoded)

	data := strings.NewReader(fmt.Sprintf(roleTmpl, role))

	err = b.client.requestWithHeader(ctx, http.MethodPost, "/auth", header, data, &resp)

	return
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Key string `json:"key"`
}
