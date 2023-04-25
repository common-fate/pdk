package client

import (
	"context"

	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/common-fate/clio/clierr"
	"github.com/common-fate/pdk/pkg/cliauth"
	"github.com/common-fate/pdk/pkg/tokenstore"
	"github.com/common-fate/provider-registry-sdk-go/pkg/registryclient"
	"github.com/common-fate/useragent"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

// ErrorHandlingClient checks the response status code
// and creates an error if the API returns greater than 300.
type ErrorHandlingClient struct {
	Client    *http.Client
	LoginHint string
}

func (rd *ErrorHandlingClient) Do(req *http.Request) (*http.Response, error) {
	// add a user agent to the request
	ua := useragent.FromContext(req.Context())
	if ua != "" {
		req.Header.Add("User-Agent", ua)
	}

	res, err := rd.Client.Do(req)
	var ne *url.Error
	if errors.As(err, &ne) && ne.Err == tokenstore.ErrNotFound {
		return nil, clierr.New(fmt.Sprintf("%s.\nTo get started with Common Fate, please run: '%s'", err, rd.LoginHint))
	}
	if err != nil {
		return nil, err
	}

	if res.StatusCode < 300 {
		// response is ok
		return res, nil
	}

	// if we get here, the API has returned an error
	// surface this as a Go error so we don't need to handle it everywhere in our CLI codebase.
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return res, errors.Wrap(err, "reading error response body")
	}

	e := clierr.New(fmt.Sprintf("Common Fate API returned an error (code %v): %s", res.StatusCode, string(body)))

	if res.StatusCode == http.StatusUnauthorized {
		e.Messages = append(e.Messages, clierr.Infof("To log in to Common Fate, run: '%s'", rd.LoginHint))
	}

	return res, e
}

// New creates a new client, specifying the URL directly.
// The client loads the OAuth2.0 tokens from the system keychain.
// The client automatically refreshes the access token if it is expired.
func NewWithAuthToken(ctx context.Context) (*registryclient.Client, error) {
	oauthConfig := cliauth.NewAuthConfig()
	var src oauth2.TokenSource

	ts := tokenstore.New()
	tok, err := ts.Token()
	if err != nil {
		if errors.Is(err, tokenstore.ErrNotFound) {
			return nil, clierr.New(fmt.Sprintf("You need to login to Common Fate Provider Registry first, please run: '%s'", "pdk login"))
		}

		return nil, err
	}

	if oauthConfig != nil {
		// if we have oauth config we can try and refresh the token automatically when it expires,
		// and save it back in the keychain.
		src = &tokenstore.NotifyRefreshTokenSource{
			New:       oauthConfig.TokenSource(ctx, tok),
			T:         tok,
			SaveToken: ts.Save,
		}
	} else {
		src = &ts
	}

	oauthClient := oauth2.NewClient(ctx, src)
	httpClient := &ErrorHandlingClient{Client: oauthClient, LoginHint: "pdk login"}
	return registryclient.New(ctx, func(co *registryclient.ClientOpts) {
		co.HTTPClient = httpClient
	})
}
