package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

	var resp AuthResponse
	err = a.client.requestWithHeader(ctx, http.MethodPost, "/refresh", header, bytes.NewReader(data), &resp)
	if err != nil {
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

	var resp AuthResponse
	err = a.client.requestWithHeader(ctx, http.MethodPost, "/auth", header, bytes.NewReader(data), &resp)
	if err != nil {
		return "", err
	}

	return resp.Key, nil
}

const (
	TokenConsumer   = "CP5tR8Zqd2txobVbWn02+YXZ6YEXDBjl8lq1cYaRHDJLE7rldjGBft5r2imUTAnExkQoSoBWSywCG93feYF5jtDN3kHpxcwKrm0Mz/JJknYZFzcZml8="
	TokenCreator    = "v2ACGxBiGJf3Jyos7MX/vq8nrp8zTVx/mT3wtGPXA/ayNBQdIZLIgd/gNlWoaS5r6AQ22zUAYa4hcbx93bKyUmIaKSJBuOGz/Sz0/dnUAZjkocF0pg=="
	TokenMaintainer = "MZhpzQUMPyNMmbrQMKcTBLzHpLpz3JB1CKuFVuHH+yqzZI6kzdjWI4OOhGw1l5NonvwZMhxTOlCmsmW2Fq/dRLgvn8EiKyOKNDsYcK8es94IwkMwKLcebw=="
	TokenAccountant = "3jpNFZwiVDAFeMEDi5tvSZ8czxgZjZr6AWaRSB0ApVueucGXpLbMVvU38HPxJtTIjEtW6BUtFb8EEkKfsw12coM+JngWNaRm9bWwJsCoG8b69oCklGK2sw=="
)

var roles = map[string]string{
	"consumer":   TokenConsumer,
	"creator":    TokenCreator,
	"maintainer": TokenMaintainer,
	"accountant": TokenAccountant,
}

func GetToken(path, method string) (string, error) {
	roleName := getRole(path, method)

	if roleName == "" {
		return "", fmt.Errorf("role not found for path '%s' and method %s", path, method)
	}

	return roles[roleName], nil
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
	{"consumer", "/bytes/*", "GET"},
	{"creator", "/bytes", "POST"},
	{"consumer", "/chunks/*", "GET"},
	{"creator", "/chunks", "POST"},
	{"consumer", "/bzz/*", "GET"},
	{"creator", "/bzz/*", "PATCH"},
	{"creator", "/bzz", "POST"},
	{"creator", "/bzz\\?*", "POST"},
	{"consumer", "/bzz/*/*", "GET"},
	{"creator", "/tags", "GET"},
	{"creator", "/tags\\?*", "GET"},
	{"creator", "/tags", "POST"},
	{"creator", "/tags/*", "(GET)|(DELETE)|(PATCH)"},
	{"creator", "/pins/*", "(GET)|(DELETE)|(POST)"},
	{"maintainer", "/pins", "GET"},
	{"creator", "/pss/send/*", "POST"},
	{"consumer", "/pss/subscribe/*", "GET"},
	{"creator", "/soc/*/*", "POST"},
	{"creator", "/feeds/*/*", "POST"},
	{"consumer", "/feeds/*/*", "GET"},
	{"maintainer", "/stamps", "GET"},
	{"maintainer", "/stamps/*", "GET"},
	{"maintainer", "/stamps/*/*", "POST"},
	{"maintainer", "/stamps/topup/*/*", "PATCH"},
	{"maintainer", "/stamps/dilute/*/*", "PATCH"},
	{"maintainer", "/addresses", "GET"},
	{"maintainer", "/blocklist", "GET"},
	{"maintainer", "/connect/*", "POST"},
	{"maintainer", "/peers", "GET"},
	{"maintainer", "/peers/*", "DELETE"},
	{"maintainer", "/pingpong/*", "POST"},
	{"maintainer", "/topology", "GET"},
	{"maintainer", "/welcome-message", "(GET)|(POST)"},
	{"maintainer", "/balances", "GET"},
	{"maintainer", "/balances/*", "GET"},
	{"maintainer", "/accounting", "GET"},
	{"maintainer", "/chequebook/cashout/*", "GET"},
	{"accountant", "/chequebook/cashout/*", "POST"},
	{"accountant", "/chequebook/withdraw", "POST"},
	{"accountant", "/chequebook/withdraw\\?*", "POST"},
	{"accountant", "/chequebook/deposit", "POST"},
	{"accountant", "/chequebook/deposit\\?*", "POST"},
	{"maintainer", "/chequebook/cheque/*", "GET"},
	{"maintainer", "/chequebook/cheque", "GET"},
	{"maintainer", "/chequebook/address", "GET"},
	{"maintainer", "/chequebook/balance", "GET"},
	{"maintainer", "/wallet", "GET"},
	{"maintainer", "/chunks/*", "(GET)|(DELETE)"},
	{"maintainer", "/reservestate", "GET"},
	{"maintainer", "/chainstate", "GET"},
	{"maintainer", "/settlements/*", "GET"},
	{"maintainer", "/settlements", "GET"},
	{"maintainer", "/transactions", "GET"},
	{"consumer", "/transactions/*", "GET"},
	{"accountant", "/transactions/*", "(POST)|(DELETE)"},
	{"consumer", "/consumed", "GET"},
	{"consumer", "/consumed/*", "GET"},
	{"consumer", "/chunks/stream", "GET"},
	{"creator", "/stewardship/*", "GET"},
	{"consumer", "/stewardship/*", "PUT"},
	{"maintainer", "/stake/*", "POST"},
	{"maintainer", "/stake", "(GET)|(DELETE)"},
}
