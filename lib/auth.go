package lib

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type AuthResponse struct {
	Iss        string
	Sub        string
	Aud        string
	Exp        string
	Given_name string
}

type Authenticator struct {
	Cache       map[string]AuthResponse
	disableAuth bool
}

func (r *AuthResponse) GetGoogleID() string {
	return r.Sub
}

func (r *AuthResponse) GetGivenName() string {
	return r.Given_name
}

func (r *AuthResponse) HasExpired(buffer int64) bool {
	expiration, _ := strconv.Atoi(r.Exp)
	return int64(expiration)+buffer < time.Now().Unix()
}

func (a *Authenticator) Initialize(disable bool) {
	a.Cache = make(map[string]AuthResponse)
	a.disableAuth = disable
}

func (a *Authenticator) Authenticate(token string) (AuthResponse, string) {
	if a.disableAuth {
		return AuthResponse{Iss: "my_iss", Sub: "my_sub", Aud: "my_aud", Exp: "my_exp", Given_name: "Ryan test"}, ""
	}

	if token == "" {
		return AuthResponse{}, "Trying to authenticate empty token."
	}

	r, cached := a.Cache[token]
	if !cached || r.HasExpired(AuthExpirationSeconds) {
		// send authentication request
		resp, err := http.Get("https://www.googleapis.com/oauth2/v3/tokeninfo?id_token=" + token)
		if err != nil {
			return AuthResponse{}, "Error received sending authentication request: " + err.Error()
		}

		// read response
		defer resp.Body.Close()
		body, readerr := ioutil.ReadAll(resp.Body)
		if readerr != nil {
			return AuthResponse{}, "Error reading authentication response: " + readerr.Error()
		}

		// unpack JSON
		b := []byte(string(body[:]))
		jsonerr := json.Unmarshal(b, &r)
		if jsonerr != nil {
			return AuthResponse{}, "Error unpacking authentication response JSON: " + jsonerr.Error()
		}

		// confirm token is valid
		if r.Aud != "168641906858-8egtsbds49ifcjgq7g6n4757q70k14h4.apps.googleusercontent.com" {
			return AuthResponse{}, "Received sign-in token for different client: " + r.Aud
		}
		if r.Iss != "accounts.google.com" && r.Iss != "https://accounts.google.com" {
			return AuthResponse{}, "Received sign-in token from different sign-in origin: " + r.Iss
		}
		if r.HasExpired(AuthExpirationSeconds) {
			return AuthResponse{}, "Received expired sign-in token."
		}

		a.Cache[token] = r
	}

	return r, ""
}
