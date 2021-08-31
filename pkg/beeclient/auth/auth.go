package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// AuthService represents Bee's Auth service
type AuthService struct {
	URL *url.URL
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Key string `json:"key"`
}

const roleTmpl = `{
    "role": "%s"
}`

// Authenticate gets the bearer security token based on given credentials
func (a *AuthService) Authenticate(ctx context.Context, role, username, password string) (resp AuthResponse, err error) {
	plain := fmt.Sprintf("%s:%s", username, password)
	encoded := base64.StdEncoding.EncodeToString([]byte(plain))

	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("Accept", "application/json")
	header.Set("Authorization", "Basic "+encoded)

	data := strings.NewReader(fmt.Sprintf(roleTmpl, role))

	fmt.Println("got URL", a.URL.String())

	if !strings.HasSuffix(a.URL.Path, "/") {
		a.URL.Path += "/"
	}

	fmt.Println("calling", a.URL)

	req, err := http.NewRequest(http.MethodPost, a.URL.String()+"/auth", data)
	if err != nil {
		return AuthResponse{}, err
	}

	req = req.WithContext(ctx)
	req.Header = header

	c := new(http.Client)
	r, err := c.Do(req)
	if err != nil {
		return AuthResponse{}, err
	}

	fmt.Println("got status code", r.StatusCode)

	defer r.Body.Close()

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("ReadAll: %w", err)
	}

	return AuthResponse{}, fmt.Errorf("real response: %s", string(b))
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
