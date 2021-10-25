package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// AuthService represents Bee's Auth service
type AuthService service

// AuthResponse represents authentication response
type AuthResponse struct {
	Key string `json:"key"`
}

func (a *AuthService) Refresh(ctx context.Context, securityToken string) (string, error) {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("Accept", "application/json")
	header.Set("Authorization", "Bearer "+securityToken)

	data, err := json.Marshal(struct {
		Expiry int `json:"expiry"`
	}{Expiry: 30})
	if err != nil {
		return "", err
	}

	r, err := a.client.httpClient.Post("/refresh", "application/json", bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	defer r.Body.Close()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("ReadAll: %w", err)
	}

	var resp AuthResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		return "", err
	}

	return resp.Key, nil
}

// Authenticate gets the bearer security token based on given credentials
func (a *AuthService) Authenticate(ctx context.Context, role, password string) (string, error) {
	plain := fmt.Sprintf("test:%s", password)
	encoded := base64.StdEncoding.EncodeToString([]byte(plain))

	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("Accept", "application/json")
	header.Set("Authorization", "Basic "+encoded)

	data, err := json.Marshal(struct {
		Role   string `json:"role"`
		Expiry int    `json:"expiry"`
	}{Role: role, Expiry: 30})
	if err != nil {
		return "", err
	}

	r, err := a.client.httpClient.Post("/auth", "application/json", bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	defer r.Body.Close()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("ReadAll: %w", err)
	}

	var resp AuthResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		return "", err
	}

	return resp.Key, nil
}

const (
	role0 = "ZwCJiyoVVOKY4fbiRw7XNiFOwqGfIXxV5MkKbI3ZaB0QYvVpmTbnEvg9pIyOCwMxgvLmnaiMlPBj+9+Q/rR86mSv3HbL17B94Mu23e+EutewXUM="
	role1 = "0vR/rRorcAsqjV9sbeOo5B8FJrXoycm96Qa57xPjN/Y4yWclQN6SrdrfMfpDb8jEWvoHC2dcOktsQXoJMLnKkLPVIyIu/0R5V/G1ZUsa3evt/Ew="
	role2 = "eM/p8zbw572azyc81tw8QnUmIan0QQM+BHC/QhKlW2e0QhxpcloDcJDPU8xunC/aSma1bVVGJeetuBbJ+ng/omgAaCi9oolL0lEyIN0oZ/v0h6Y="
	role3 = "2HgybgRX8FFuTkGHj0XGHeIbpzwnfjlJmQD/rbmSxgE399Gz42kAEbPMGtcd3fqF+SOzPOpOg/jv1bDHE1C6fiW4xzf7lEEa6CEenkiTF6e0p3U="
)

func GetToken(path, method string) string {
	m := map[string]string{
		"role0": role0,
		"role1": role1,
		"role2": role2,
		"role3": role3,
	}

	r := getRole(path, method)

	return m[r]
}

func getRole(path, method string) string {
	for _, v := range policies {
		if v[2] != method {
			if !strings.Contains(v[2], fmt.Sprintf("(%s)", method)) {
				continue
			}
		}
		re := regexp.MustCompile(v[1])
		if re.Match([]byte(path)) {
			return v[0]
		}
	}

	return ""
}

var policies = [][]string{
	{"role0", "/bytes/*", "GET"},
	{"role1", "/bytes", "POST"},
	{"role0", "/chunks/*", "GET"},
	{"role1", "/chunks", "POST"},
	{"role0", "/bzz/*", "GET"},
	{"role1", "/bzz/*", "PATCH"},
	{"role1", "/bzz", "POST"},
	{"role0", "/bzz/*/*", "GET"},
	{"role1", "/tags", "(GET)|(POST)"},
	{"role1", "/tags/*", "(GET)|(DELETE)|(PATCH)"},
	{"role1", "/pins/*", "(GET)|(DELETE)|(POST)"},
	{"role2", "/pins", "GET"},
	{"role1", "/pss/send/*", "POST"},
	{"role0", "/pss/subscribe/*", "GET"},
	{"role1", "/soc/*/*", "POST"},
	{"role1", "/feeds/*/*", "POST"},
	{"role0", "/feeds/*/*", "GET"},
	{"role2", "/stamps", "GET"},
	{"role2", "/stamps/*", "GET"},
	{"role2", "/stamps/*/*", "POST"},
	{"role2", "/addresses", "GET"},
	{"role2", "/blocklist", "GET"},
	{"role2", "/connect/*", "POST"},
	{"role2", "/peers", "GET"},
	{"role2", "/peers/*", "DELETE"},
	{"role2", "/pingpong/*", "POST"},
	{"role2", "/topology", "GET"},
	{"role2", "/welcome-message", "(GET)|(POST)"},
	{"role2", "/balances", "GET"},
	{"role2", "/balances/*", "GET"},
	{"role2", "/chequebook/cashout/*", "GET"},
	{"role3", "/chequebook/cashout/*", "POST"},
	{"role3", "/chequebook/withdraw", "POST"},
	{"role3", "/chequebook/deposit", "POST"},
	{"role2", "/chequebook/cheque/*", "GET"},
	{"role2", "/chequebook/cheque", "GET"},
	{"role2", "/chequebook/address", "GET"},
	{"role2", "/chequebook/balance", "GET"},
	{"role2", "/chunks/*", "(GET)|(DELETE)"},
	{"role2", "/reservestate", "GET"},
	{"role2", "/chainstate", "GET"},
	{"role2", "/settlements/*", "GET"},
	{"role2", "/settlements", "GET"},
	{"role2", "/transactions", "GET"},
	{"role0", "/transactions/*", "GET"},
	{"role3", "/transactions/*", "(POST)|(DELETE)"},
	{"role0", "/consumed", "GET"},
	{"role0", "/consumed/*", "GET"},
	{"role0", "/chunks/stream", "GET"},
	{"role0", "/stewardship/*", "PUT"},
}
