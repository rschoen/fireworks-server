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

func (r *AuthResponse) GetGoogleID() string {
	return r.Sub
}

func (r *AuthResponse) GetGivenName() string {
	return r.Given_name
}

func Authenticate(token string) (AuthResponse, string) {
	if token == "" {
		return AuthResponse{}, "Trying to authenticate empty token."
	}

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
	var r AuthResponse
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
	expiration, _ := strconv.Atoi(r.Exp)
	if int64(expiration) < time.Now().Unix() {
		return AuthResponse{}, "Received expired sign-in token."
	}

	return r, ""
}
