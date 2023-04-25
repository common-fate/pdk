package cliauth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

func NewAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID: "77sr88lvofdg37lptf4r402nf1",
		Scopes:   []string{"openid", "email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://common-fate-registry.auth.us-west-2.amazoncognito.com/oauth2/authorize",
			TokenURL: "https://common-fate-registry.auth.us-west-2.amazoncognito.com/oauth2/token",
		},
		RedirectURL: "http://localhost:8848/oauth/login/callback",
	}
}

func generateStateOauthCookie(w http.ResponseWriter) string {
	var expiration = time.Now().Add(20 * time.Minute)

	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// shouldn't happen
		panic(err)
	}
	state := base64.URLEncoding.EncodeToString(b)
	cookie := http.Cookie{Name: "oauthstate", Value: state, Expires: expiration}
	http.SetCookie(w, &cookie)

	return state
}
