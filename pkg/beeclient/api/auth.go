package api

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// AuthService represents Bee's Auth service
type AuthService service

// AuthResponse represents authentication response
type AuthResponse struct {
	Key string `json:"key"`
}

const roleTmpl = `{"role": "%s"}`

// Authenticate gets the bearer security token based on given credentials
func (b *AuthService) Authenticate(ctx context.Context, url, role, username, password string) (resp AuthResponse, err error) {
	plain := fmt.Sprintf("%s:%s", username, password)
	encoded := base64.StdEncoding.EncodeToString([]byte(plain))

	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("Accept", "application/json")
	header.Set("Authorization", "Basic "+encoded)

	body := strings.NewReader(fmt.Sprintf(roleTmpl, role))

	var netTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Dial: (&net.Dialer{
			Timeout: 15 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 15 * time.Second,
	}

	var client = &http.Client{
		Timeout:   time.Second * 15,
		Transport: netTransport,
	}

	fmt.Println("GET ", url+"health")

	req, err := http.NewRequest(http.MethodGet, url+"health", body)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("new request: %w", err)
	}
	req = req.WithContext(ctx)
	req.Header = header

	res, err := client.Do(req)

	if err != nil {
		return AuthResponse{}, fmt.Errorf("exec request: %w", err)
	}

	if res.StatusCode != http.StatusCreated {
		return AuthResponse{}, fmt.Errorf("bad status: %d", res.StatusCode)
	}

	bts, _ := ioutil.ReadAll(res.Body)

	fmt.Println("got body", string(bts))
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return AuthResponse{}, fmt.Errorf("decoder: %w", err)
	}

	return
}

func GetRole(path, method string) string {
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